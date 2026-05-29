package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AgentPal/AgentCRM/internal/i18n"
	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/spf13/cobra"
)

// eventsCmd 管理事件订阅。
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: i18n.T("cmd.events.short"),
}

var eventsPollCmd = &cobra.Command{
	Use:   "poll",
	Short: i18n.T("cmd.events.poll.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		as, _ := cmd.Flags().GetString("as")
		filter, _ := cmd.Flags().GetString("filter")
		limit, _ := cmd.Flags().GetInt("limit")
		includeSelf, _ := cmd.Flags().GetBool("include-self")

		if as == "" {
			as = getActor()
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		// 读取 subscriber cursor
		sc, err := s.FS.ReadSubscriberCursor(as)
		if err != nil {
			return fmt.Errorf("read cursor: %w", err)
		}

		// 防循环：默认排除自己
		excludeActor := ""
		if !includeSelf {
			excludeActor = as
		}

		events, err := s.Events.Poll(sc.LastSeq, filter, excludeActor, limit)
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.Marshal(events)
			fmt.Println(string(out))
		} else {
			if len(events) == 0 {
				fmt.Println(i18n.T("output.events.poll.none"))
				return nil
			}
			for _, e := range events {
				fmt.Printf("#%d [%s] %s by %s\n", e.Seq, e.TS[:16], e.Type, e.Actor)
			}
			fmt.Print(i18n.T("output.events.poll.count", len(events), sc.LastSeq))
		}
		return nil
	},
}

var eventsAckCmd = &cobra.Command{
	Use:   "ack",
	Short: i18n.T("cmd.events.ack.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		as, _ := cmd.Flags().GetString("as")
		upToSeq, _ := cmd.Flags().GetInt64("up-to-seq")

		if as == "" {
			as = getActor()
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		sc, err := s.FS.ReadSubscriberCursor(as)
		if err != nil {
			return err
		}

		sc.LastSeq = upToSeq

		if err := s.FS.WriteSubscriberCursor(sc); err != nil {
			return err
		}

		fmt.Println(i18n.T("output.events.ack.ok", as, upToSeq))
		return nil
	},
}

var eventsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: i18n.T("cmd.events.watch.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		as, _ := cmd.Flags().GetString("as")
		filter, _ := cmd.Flags().GetString("filter")
		interval, _ := cmd.Flags().GetInt("interval")

		if as == "" {
			as = getActor()
		}
		if interval <= 0 {
			interval = 60
		}

		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		sc, err := s.FS.ReadSubscriberCursor(as)
		if err != nil {
			return fmt.Errorf("read cursor: %w", err)
		}

		fmt.Println(i18n.T("output.events.watch.start", as, interval, sc.LastSeq))
		fmt.Println(i18n.T("output.events.watch.stop"))

		for {
			events, err := s.Events.Poll(sc.LastSeq, filter, as, interval)
			if err != nil {
				return fmt.Errorf("poll: %w", err)
			}

			for _, e := range events {
				fmt.Printf("#%d [%s] %s by %s  payload=%s\n", e.Seq, e.TS[:16], e.Type, e.Actor, e.Payload)
				if e.Seq > sc.LastSeq {
					sc.LastSeq = e.Seq
				}
			}

			if len(events) > 0 {
				s.FS.WriteSubscriberCursor(sc)
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}
	},
}

var subscribersCmd = &cobra.Command{
	Use:   "subscribers",
	Short: i18n.T("cmd.events.subscribers.short"),
}

var subscribersListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("cmd.events.subscribers.list.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		dir := filepath.Join(s.ConfigDir(), ".subscribers")
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println(i18n.T("output.events.subscribers.none"))
				return nil
			}
			return fmt.Errorf("read subscribers dir: %w", err)
		}

		var subscribers []*model.SubscriberCursor
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			actor := strings.TrimSuffix(e.Name(), ".json")
			sc, err := s.FS.ReadSubscriberCursor(actor)
			if err != nil {
				continue
			}
			subscribers = append(subscribers, sc)
		}

		if format == "json" {
			out, _ := json.Marshal(subscribers)
			fmt.Println(string(out))
		} else {
			if len(subscribers) == 0 {
				fmt.Println(i18n.T("output.events.subscribers.none"))
				return nil
			}
			for _, sc := range subscribers {
				checkAt := sc.LastCheck
				if checkAt == "" {
					checkAt = "-"
				}
				fmt.Printf("%s  cursor: %d  last_check: %s", sc.Actor, sc.LastSeq, checkAt)
				if sc.Filter != "" {
					fmt.Printf("  filter: %s", sc.Filter)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var subscribersResetCmd = &cobra.Command{
	Use:   "reset <actor>",
	Short: i18n.T("cmd.events.subscribers.reset.short"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		sc := &model.SubscriberCursor{
			Actor:   args[0],
			LastSeq: 0,
		}
		if err := s.FS.WriteSubscriberCursor(sc); err != nil {
			return err
		}
		fmt.Println(i18n.T("output.events.subscribers.reset.ok", args[0]))
		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsPollCmd)
	eventsCmd.AddCommand(eventsAckCmd)
	eventsCmd.AddCommand(eventsWatchCmd)
	eventsCmd.AddCommand(subscribersCmd)

	subscribersCmd.AddCommand(subscribersListCmd)
	subscribersCmd.AddCommand(subscribersResetCmd)

	eventsPollCmd.Flags().String("as", "", i18n.T("flag.events.as"))
	eventsPollCmd.Flags().String("filter", "", i18n.T("flag.events.filter"))
	eventsPollCmd.Flags().Int("limit", 100, i18n.T("flag.events.limit"))
	eventsPollCmd.Flags().Bool("include-self", false, i18n.T("flag.events.include_self"))

	eventsAckCmd.Flags().String("as", "", i18n.T("flag.events.as"))
	eventsAckCmd.Flags().Int64("up-to-seq", 0, i18n.T("flag.events.up_to_seq"))

	eventsWatchCmd.Flags().String("as", "", i18n.T("flag.events.as"))
	eventsWatchCmd.Flags().String("filter", "", i18n.T("flag.events.filter"))
	eventsWatchCmd.Flags().Int("interval", 60, i18n.T("flag.events.interval"))
}
