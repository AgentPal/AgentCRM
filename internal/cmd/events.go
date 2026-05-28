package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AgentPal/AgentCRM/internal/model"
	"github.com/spf13/cobra"
)

// eventsCmd 管理事件订阅。
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "管理事件订阅（多 Agent 协作）",
}

var eventsPollCmd = &cobra.Command{
	Use:   "poll",
	Short: "拉取新事件",
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
				fmt.Println("无新事件")
				return nil
			}
			for _, e := range events {
				fmt.Printf("#%d [%s] %s by %s\n", e.Seq, e.TS[:16], e.Type, e.Actor)
			}
			fmt.Printf("\n共 %d 个事件 (cursor: %d)\n", len(events), sc.LastSeq)
		}
		return nil
	},
}

var eventsAckCmd = &cobra.Command{
	Use:   "ack",
	Short: "推进事件 cursor",
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

		fmt.Printf("已推进 %s cursor 到 #%d\n", as, upToSeq)
		return nil
	},
}

var eventsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "持续监听新事件（阻塞式轮询）",
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

		fmt.Printf("开始监听事件 (actor: %s, interval: %ds, cursor: %d)\n", as, interval, sc.LastSeq)
		fmt.Println("按 Ctrl+C 停止")

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
	Short: "管理订阅者",
}

var subscribersListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有订阅者",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := getStore()
		if err != nil {
			return err
		}
		defer s.Close()

		dir := filepath.Join(s.ConfigDir(), "subscribers")
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("无订阅者")
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
				fmt.Println("无订阅者")
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
	Short: "重置订阅者 cursor",
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
		fmt.Printf("已重置 %s cursor\n", args[0])
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

	eventsPollCmd.Flags().String("as", "", "订阅者名称")
	eventsPollCmd.Flags().String("filter", "", "事件过滤器")
	eventsPollCmd.Flags().Int("limit", 100, "返回数量上限")
	eventsPollCmd.Flags().Bool("include-self", false, "包含自己产生的事件")

	eventsAckCmd.Flags().String("as", "", "订阅者名称")
	eventsAckCmd.Flags().Int64("up-to-seq", 0, "推进到此 seq")

	eventsWatchCmd.Flags().String("as", "", "订阅者名称")
	eventsWatchCmd.Flags().String("filter", "", "事件过滤器")
	eventsWatchCmd.Flags().Int("interval", 60, "轮询间隔（秒）")
}
