// Package gcalendar provides enhanced Google Calendar functionality with smart event notifications.
// It extends the basic Google Calendar API with features like duplicate notification prevention,
// smart timing-based alerts, and comprehensive event management.
package gcalendar

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gmail-watcher/io_helpers"
	"google.golang.org/api/calendar/v3"
)

// CalendarEvent represents a processed calendar event with rich metadata.
// It contains all the information needed for intelligent notifications and display.
type CalendarEvent struct {
	ID           string    // Unique event identifier from Google Calendar
	Title        string    // Event title/summary
	Description  string    // Event description/body text
	StartTime    time.Time // Event start time (local timezone)
	EndTime      time.Time // Event end time (local timezone)
	CalendarID   string    // ID of the calendar containing this event
	CalendarName string    // Human-readable calendar name
	CreatorEmail string    // Email of the event creator
	Location     string    // Event location (if specified)
	AllDay       bool      // True if this is an all-day event
}

// CalendarService wraps the Google Calendar service with enhanced functionality.
// It provides smart notifications, duplicate prevention, and concurrent-safe operations.
type CalendarService struct {
	Service        *calendar.Service    // Underlying Google Calendar API service
	NotifiedEvents map[string]time.Time // Track notified events to avoid duplicates
	mutex          sync.RWMutex         // Protect concurrent access to NotifiedEvents
}

// NewCalendarService creates a new enhanced calendar service with notification tracking.
// It initializes the service with thread-safe notification deduplication capabilities.
//
// Parameters:
//   - srv: The underlying Google Calendar API service
//
// Returns:
//   - *CalendarService: A new enhanced calendar service instance
func NewCalendarService(srv *calendar.Service) *CalendarService {
	return &CalendarService{
		Service:        srv,
		NotifiedEvents: make(map[string]time.Time),
	}
}

// dateEqual checks if two dates are on the same day
func dateEqual(date1 time.Time, date2 time.Time) bool {
	year1, month1, day1 := date1.Date()
	year2, month2, day2 := date2.Date()
	return year1 == year2 && month1 == month2 && day1 == day2
}

// GetEvents retrieves and displays events (legacy function for compatibility)
func GetEvents(srv *calendar.Service, maxResults int64) error {
	calSrv := NewCalendarService(srv)
	events, err := calSrv.GetUpcomingEvents(maxResults, time.Now(), time.Now().AddDate(0, 0, 7))
	if err != nil {
		return fmt.Errorf("failed to get upcoming events: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No upcoming events found.")
		return nil
	}

	// Group events by calendar
	calendarEvents := make(map[string][]*CalendarEvent)
	for _, event := range events {
		calendarEvents[event.CalendarName] = append(calendarEvents[event.CalendarName], event)
	}

	// Display events grouped by calendar
	for calName, calEvents := range calendarEvents {
		fmt.Printf("\n📅 Calendar: %s\n", calName)
		fmt.Println(strings.Repeat("-", 50))
		
		for _, event := range calEvents {
			timeStr := event.StartTime.Format("Mon, Jan 2 at 3:04 PM")
			if event.AllDay {
				timeStr = event.StartTime.Format("Mon, Jan 2 (All Day)")
			}
			
			fmt.Printf("🕒 %s\n", timeStr)
			fmt.Printf("📝 %s\n", event.Title)
			if event.Description != "" {
				fmt.Printf("📄 %s\n", event.Description)
			}
			if event.Location != "" {
				fmt.Printf("📍 %s\n", event.Location)
			}
			fmt.Println()
			
			// Send notification for today's events
			if dateEqual(event.StartTime, time.Now()) {
				_ = io_helpers.Notify(event.Title, fmt.Sprintf("Today at %s", event.StartTime.Format("3:04 PM")))
			}
		}
	}
	return nil
}

// GetUpcomingEvents retrieves upcoming events within a time range
func (cs *CalendarService) GetUpcomingEvents(maxResults int64, startTime, endTime time.Time) ([]*CalendarEvent, error) {
	var allEvents []*CalendarEvent
	
	calList, err := cs.Service.CalendarList.List().Do()
	if err != nil {
		log.Printf("Error getting calendar list: %s", err)
		return nil, fmt.Errorf("failed to get calendar list: %w", err)
	}
	
	for _, calItem := range calList.Items {
		events, err := cs.Service.Events.List(calItem.Id).ShowDeleted(false).
			SingleEvents(true).MaxResults(maxResults).OrderBy("startTime").
			TimeMin(startTime.Format(time.RFC3339)).TimeMax(endTime.Format(time.RFC3339)).Do()
		if err != nil {
			log.Printf("Error getting events for calendar %s: %v", calItem.Summary, err)
			continue
		}
		
		for _, item := range events.Items {
			event, err := cs.parseEvent(item, calItem)
			if err != nil {
				log.Printf("Error parsing event %s: %v", item.Id, err)
				continue
			}
			allEvents = append(allEvents, event)
		}
	}
	
	// Sort events by start time
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].StartTime.Before(allEvents[j].StartTime)
	})
	
	return allEvents, nil
}

// parseEvent converts a Google Calendar event to our CalendarEvent struct
func (cs *CalendarService) parseEvent(item *calendar.Event, calItem *calendar.CalendarListEntry) (*CalendarEvent, error) {
	event := &CalendarEvent{
		ID:           item.Id,
		Title:        item.Summary,
		Description:  item.Description,
		CalendarID:   calItem.Id,
		CalendarName: calItem.Summary,
		Location:     item.Location,
	}
	
	if item.Creator != nil {
		event.CreatorEmail = item.Creator.Email
	}
	
	// Parse start time
	var err error
	if item.Start.DateTime != "" {
		event.StartTime, err = time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start datetime %s: %w", item.Start.DateTime, err)
		}
	} else if item.Start.Date != "" {
		// All-day event
		event.StartTime, err = time.Parse("2006-01-02", item.Start.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start date %s: %w", item.Start.Date, err)
		}
		event.AllDay = true
	}
	
	// Parse end time
	if item.End.DateTime != "" {
		event.EndTime, err = time.Parse(time.RFC3339, item.End.DateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end datetime %s: %w", item.End.DateTime, err)
		}
	} else if item.End.Date != "" {
		event.EndTime, err = time.Parse("2006-01-02", item.End.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end date %s: %w", item.End.Date, err)
		}
	}
	
	return event, nil
}

// CheckUpcomingEvents checks for events that need notifications
func (cs *CalendarService) CheckUpcomingEvents(notifyMinutesBefore []int) error {
	now := time.Now()
	endTime := now.AddDate(0, 0, 1) // Check next 24 hours
	
	events, err := cs.GetUpcomingEvents(50, now, endTime)
	if err != nil {
		return fmt.Errorf("failed to get upcoming events: %w", err)
	}
	
	for _, event := range events {
		for _, minutesBefore := range notifyMinutesBefore {
			notifyTime := event.StartTime.Add(-time.Duration(minutesBefore) * time.Minute)
			
			// Check if we should notify now (within 1 minute window)
			if now.After(notifyTime) && now.Before(notifyTime.Add(time.Minute)) {
				// Check if we've already notified for this time
				notifyKey := fmt.Sprintf("%s_%d", event.ID, minutesBefore)
				
				cs.mutex.RLock()
				lastNotified, exists := cs.NotifiedEvents[notifyKey]
				cs.mutex.RUnlock()
				
				if !exists || now.Sub(lastNotified) > time.Hour {
					cs.sendEventNotification(event, minutesBefore)
					
					cs.mutex.Lock()
					cs.NotifiedEvents[notifyKey] = now
					cs.mutex.Unlock()
				}
			}
		}
	}
	
	return nil
}

// sendEventNotification sends a notification for an upcoming event
func (cs *CalendarService) sendEventNotification(event *CalendarEvent, minutesBefore int) {
	var title, message string
	
	if minutesBefore == 0 {
		title = "📅 Event Starting Now"
		message = fmt.Sprintf("%s\n🕒 %s", event.Title, event.StartTime.Format("3:04 PM"))
	} else {
		title = fmt.Sprintf("📅 Event in %d minutes", minutesBefore)
		message = fmt.Sprintf("%s\n🕒 %s", event.Title, event.StartTime.Format("3:04 PM"))
	}
	
	if event.Location != "" {
		message += fmt.Sprintf("\n📍 %s", event.Location)
	}
	
	log.Printf("Sending calendar notification: %s - %s", title, event.Title)
	_ = io_helpers.Notify(message, title)
}

// GetTodaysEvents gets all events for today
func (cs *CalendarService) GetTodaysEvents() ([]*CalendarEvent, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	return cs.GetUpcomingEvents(100, startOfDay, endOfDay)
}

// GetEventsInRange gets events within a specific time range
func (cs *CalendarService) GetEventsInRange(startTime, endTime time.Time, maxResults int64) ([]*CalendarEvent, error) {
	return cs.GetUpcomingEvents(maxResults, startTime, endTime)
}

// RunCalendarDaemon runs a continuous calendar notification service.
// It periodically checks for upcoming events and sends notifications at configured intervals.
// This function runs indefinitely until the process is terminated.
//
// Parameters:
//   - checkIntervalMinutes: How often to check for upcoming events (in minutes)
//   - notifyMinutesBefore: Array of minutes before events to send notifications
//
// Returns:
//   - error: Any error that caused the daemon to stop (currently unreachable)
//
// Example usage:
//   // Send notifications 30, 15, 5, and 0 minutes before events, checking every 2 minutes
//   daemon.RunCalendarDaemon(2, []int{30, 15, 5, 0})
func (cs *CalendarService) RunCalendarDaemon(checkIntervalMinutes int, notifyMinutesBefore []int) error {
	log.Printf("Starting calendar daemon with check interval: %d minutes", checkIntervalMinutes)
	log.Printf("Notification triggers: %v minutes before events", notifyMinutesBefore)
	
	// Set up periodic ticker for checking events
	ticker := time.NewTicker(time.Duration(checkIntervalMinutes) * time.Minute)
	defer ticker.Stop()
	
	// Perform initial check immediately on startup
	if err := cs.CheckUpcomingEvents(notifyMinutesBefore); err != nil {
		log.Printf("Error in initial calendar check: %v", err)
	}
	
	// Main daemon loop - runs indefinitely
	for {
		select {
		case <-ticker.C:
			// Check for events that need notifications
			if err := cs.CheckUpcomingEvents(notifyMinutesBefore); err != nil {
				log.Printf("Error checking upcoming events: %v", err)
			}
		}
	}
}