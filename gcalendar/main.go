package gcalendar

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"
)

func GetEvents(srv *calendar.Service, max_results int64) {

	t := time.Now().Format(time.RFC3339)
	cal_list, err := srv.CalendarList.List().Do()
	if err != nil {
		fmt.Printf("Error occured %s", err)
		log.Fatalf("Error occcured invalid dereference %v", err)
	}
	for _, cal_name := range cal_list.Items {
		fmt.Printf("Calendar := %v", cal_name.Summary)

		events, err := srv.Events.List(cal_name.Id).ShowDeleted(false).
			SingleEvents(true).MaxResults(max_results).OrderBy("startTime").TimeMin(t).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}
		if len(events.Items) == 0 {
			fmt.Println("No upcoming events found.")
		} else {
			for _, item := range events.Items {
				date := item.Start.DateTime
				if date == "" {
					date = item.Start.Date
				}
				fmt.Printf("%v (%v)\n", item.Summary, date)
			}
		}
	}
}
