package gcalendar

import (
	"fmt"
	"github.com/gmail-watcher/io_helpers"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"
)

func date_equal(date1 time.Time, date2 time.Time) bool {
	year1, month1, day1 := date1.Date()
	year2, month2, day2 := date2.Date()
	return year1 == year2 && month1 == month2 && day1 == day2
}
func GetEvents(srv *calendar.Service, max_results int64) error {

	today := time.Now()
	t := time.Now().Format(time.RFC3339)
	cal_list, err := srv.CalendarList.List().Do()
	if err != nil {
		log.Printf("Error occured %s", err)
		return fmt.Errorf("error occcured invalid dereference %w", err)
	}
	for _, cal_name := range cal_list.Items {
		fmt.Printf("Calendar := %v\n", cal_name.Summary)
		events, err := srv.Events.List(cal_name.Id).ShowDeleted(false).
			SingleEvents(true).MaxResults(max_results).OrderBy("startTime").TimeMin(t).Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve next ten of the user's events: %w", err)
		}
		if len(events.Items) == 0 {
			fmt.Println("No upcoming events found.")
		} else {
			for _, item := range events.Items {
				date := item.Start.DateTime
				datetime_val, err := time.Parse(time.RFC3339, item.Start.DateTime)
				if err != nil {
					return fmt.Errorf("error parsing as datetime %v for value %w", date, err)
				} else {
					if date_equal(datetime_val, today) {
						_ = io_helpers.Notify(item.Summary, item.Creator.Email)
					}

				}
				fmt.Printf("%v :- %v (%s)\n", item.Description, item.Summary, date)
			}
		}
	}
	return nil
}
