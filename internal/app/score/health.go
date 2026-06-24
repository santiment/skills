package score

import (
	"time"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/clients/arena"
	"santiment.net/san-skills/internal/clients/sanr"
	"santiment.net/san-skills/internal/platform/exitcode"
)

type backendHealth struct {
	OK         bool   `json:"ok"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	LatencyMs  int64  `json:"latencyMs"`
	Error      string `json:"error,omitempty"`
}

type healthReport struct {
	OK       bool                     `json:"ok"`
	Backends map[string]backendHealth `json:"backends"`
}

func newHealthCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "health",
		Aliases: []string{"ping"},
		Short:   "Check connectivity to both backends (Sanr and Arena)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := app.ctx(cmd)
			report := healthReport{Backends: map[string]backendHealth{}}

			report.Backends[sanr.Backend] = probe(func() (int, error) {
				c, err := app.sanr()
				if err != nil {
					return 0, err
				}
				resp, err := c.GetV1PingWithResponse(ctx)
				if err != nil {
					return 0, err
				}
				return resp.StatusCode(), nil
			})

			report.Backends[arena.Backend] = probe(func() (int, error) {
				c, err := app.arena()
				if err != nil {
					return 0, err
				}
				resp, err := c.PingControllerPingWithResponse(ctx)
				if err != nil {
					return 0, err
				}
				return resp.StatusCode(), nil
			})

			report.OK = true
			overall := exitcode.OK
			for _, h := range report.Backends {
				if !h.OK {
					report.OK = false
					if overall == exitcode.OK {
						if h.HTTPStatus > 0 {
							overall = exitcode.FromHTTPStatus(h.HTTPStatus)
						} else {
							overall = exitcode.Network
						}
					}
				}
			}

			if err := app.printer.EmitValue(report); err != nil {
				return app.fail(err)
			}
			app.exitCode = overall
			return nil
		},
	}
}

// probe runs a single backend ping, capturing latency, status, and errors.
func probe(call func() (int, error)) backendHealth {
	start := time.Now()
	status, err := call()
	latency := time.Since(start).Milliseconds()
	h := backendHealth{LatencyMs: latency, HTTPStatus: status}
	if err != nil {
		h.Error = err.Error()
		return h
	}
	h.OK = status >= 200 && status < 300
	return h
}
