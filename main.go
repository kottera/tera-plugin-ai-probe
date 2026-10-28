// Command ai-probe is the plugin-ai-authority dogfood round's probe. It
// serves one tool, check_caller, to its own types' sessions, and a UI whose
// routes under /probe/ act for the signed-in person or as the plugin itself.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/kottera/sdk/plugin/pluginsdk"
	"github.com/kottera/sdk/plugin/start"
	"github.com/kottera/sdk/plugin/webui"
)

// pluginName is the name manifest.hcl's plugin_manifest block carries.
const pluginName = "ai-probe"

func main() {
	c, err := start.Read()
	if err != nil {
		fail(err)
	}
	tera := pluginsdk.NewTera(c.Socket)
	app, err := fs.Sub(dist, "web/dist")
	if err != nil {
		fail(err)
	}
	kit := webui.App(app,
		webui.WithTera(tera),
		// The verbs the probe invokes as itself, under its ends grant:
		// cancel, conclude (a transition) and interrupt (a guide).
		webui.WithPluginVerbs("work_order/cancel", "work_order/transition", "work_order/guide"),
	)
	calls := &callLog{path: c.Data + "/calls.jsonl"}
	probeRoutes(kit, c, calls)
	p, err := pluginsdk.New(c, pluginsdk.Options{Name: pluginName, Version: "0.2.0", UI: kit})
	if err != nil {
		fail(err)
	}
	if err := p.Register(checkCaller(calls)); err != nil {
		fail(err)
	}
	p.Ready()
	if err := pluginsdk.RunPlugin(context.Background(), p); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
