package log_loop_test

import (
	"github.com/floriansw/go-hll-rcon/log_loop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

const (
	connected    = "[355 ms (1671484269)] CONNECTED [1.Fjg]ToastyMcToast (76561198025480905)"
	disconnected = "[9.33 sec (1671484260)] DISCONNECTED One (76561198032765590)"
	kill         = "[1:49 min (1671484160)] KILL: [1.Fjg]ToastyMcToast(Axis/76561198025480905) -> Spinning B(Allies/76561198024946722) with M3 GREASE GUN"
	chat         = "[52.6 sec (1671484602)] CHAT[Unit][chiefjustice10(Axis/76561198076714203)]: gg hat semi viel Spaß gemacht :D"
	matchStart   = "[4.01 sec (1737300987)] MATCH START CARENTAN Skirmish "
	matchEnd     = "[4.01 sec (1737300987)] MATCH ENDED `ST MARIE DU MONT Warfare` ALLIED (2 - 3) AXIS "
)

var _ = Describe("", func() {
	It("parses CONNECTED message", func() {
		l, err := log_loop.ParseLogLine(connected)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       connected,
			Timestamp: time.Unix(1671484269, 0),
			Action:    log_loop.ActionConnected,
			Actor: log_loop.Player{
				Name:      "[1.Fjg]ToastyMcToast",
				SteamId64: "76561198025480905",
			},
			Subject: log_loop.Player{},
			Weapon:  "",
			Message: "",
			Rest:    "",
		}))
	})

	It("parses DISCONNECTED message", func() {
		l, err := log_loop.ParseLogLine(disconnected)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       disconnected,
			Timestamp: time.Unix(1671484260, 0),
			Action:    log_loop.ActionDisconnected,
			Actor: log_loop.Player{
				Name:      "One",
				SteamId64: "76561198032765590",
			},
			Subject: log_loop.Player{},
			Weapon:  "",
			Message: "",
			Rest:    "",
		}))
	})

	It("parses KILL message", func() {
		l, err := log_loop.ParseLogLine(kill)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       kill,
			Timestamp: time.Unix(1671484160, 0),
			Action:    log_loop.ActionKill,
			Actor: log_loop.Player{
				Name:      "[1.Fjg]ToastyMcToast",
				SteamId64: "76561198025480905",
				Team:      "axis",
			},
			Subject: log_loop.Player{
				Name:      "Spinning B",
				SteamId64: "76561198024946722",
				Team:      "allies",
			},
			Weapon:  "M3 GREASE GUN",
			Message: "",
			Rest:    "",
		}))
	})

	It("parses CHAT message", func() {
		l, err := log_loop.ParseLogLine(chat)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       chat,
			Timestamp: time.Unix(1671484602, 0),
			Action:    log_loop.ActionChat,
			Actor: log_loop.Player{
				Name:      "chiefjustice10",
				SteamId64: "76561198076714203",
				Team:      "axis",
			},
			Message: "gg hat semi viel Spaß gemacht :D",
			Rest:    "Unit",
		}))
	})

	It("parses MATCH START message", func() {
		l, err := log_loop.ParseLogLine(matchStart)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       matchStart,
			Timestamp: time.Unix(1737300987, 0),
			Action:    log_loop.ActionMatchStart,
			Actor:     log_loop.Player{},
			Message:   "CARENTAN Skirmish",
			Rest:      "",
		}))
	})

	It("parses MATCH END message", func() {
		l, err := log_loop.ParseLogLine(matchEnd)
		Expect(err).ToNot(HaveOccurred())

		Expect(l).To(Equal(log_loop.StructuredLogLine{
			Raw:       matchEnd,
			Timestamp: time.Unix(1737300987, 0),
			Action:    log_loop.ActionMatchEnded,
			Actor:     log_loop.Player{},
			Message:   "ST MARIE DU MONT Warfare",
			Result: &log_loop.MatchResult{
				Axis:   3,
				Allied: 2,
			},
			Rest: "",
		}))
	})
})
