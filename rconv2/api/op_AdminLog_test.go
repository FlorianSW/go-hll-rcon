package api_test

import (
	"github.com/floriansw/go-hll-rcon/rconv2/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

const (
	connected = "[355 ms (1743938197)] CONNECTED [1.Fjg]ToastyMcToast (76561198025480905)"
)

var _ = Describe("", func() {
	It("parses event received timestamp", func() {
		l := api.AdminLogEntry{Timestamp: "2025.04.06-15:24:23:369", Message: ""}

		Expect(l.ReceivedTime()).To(BeTemporally("~", time.Date(2025, 4, 6, 15, 24, 23, 369000000, time.UTC)))
	})

	It("parses event time", func() {
		l := api.AdminLogEntry{Timestamp: "2025.04.06-15:24:23:369", Message: connected}

		Expect(l.EventTime()).To(BeTemporally("~", time.Date(2025, 4, 6, 13, 16, 37, 0, time.Local)))
	})
})
