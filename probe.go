package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kottera/sdk/plugin/pluginsdk"
	"github.com/kottera/sdk/plugin/start"
	"github.com/kottera/sdk/plugin/webui"
	"github.com/kottera/sdk/workorder"
)

// callLog keeps every caller assertion check_caller verified, one JSON line
// each, in the plugin's data directory.
type callLog struct {
	mu   sync.Mutex
	path string
}

func (l *callLog) add(entry map[string]any) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func (l *callLog) read() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, err := os.ReadFile(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

// checkCaller is the served tool the probe's types grant. It answers with
// the caller assertion tera attached, as the SDK verified it, and records it.
func checkCaller(calls *callLog) pluginsdk.Tool {
	return pluginsdk.Tool{
		Name:        "check_caller",
		Description: "Answers with the caller assertion tera attached to this call: who it is for, and the Work Order, dispatch, worker and type it came from.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"note":{"type":"string"}}}`),
		Handler: func(ctx context.Context, call pluginsdk.Call) (*pluginsdk.Result, error) {
			var in struct {
				Note string `json:"note"`
			}
			_ = json.Unmarshal(call.Arguments, &in)
			caller, ok := pluginsdk.VerifiedCaller(ctx)
			entry := map[string]any{"at": time.Now().UTC().Format(time.RFC3339Nano), "note": in.Note, "verified": ok}
			if ok {
				entry["claims"] = caller.Claims
			}
			if err := calls.add(entry); err != nil {
				return nil, err
			}
			return pluginsdk.StructuredResult(entry), nil
		},
	}
}

// probeRequest is the body of every POST under /probe/.
type probeRequest struct {
	Type    string          `json:"type"`
	Worker  string          `json:"worker"`
	Title   string          `json:"title"`
	Inputs  json.RawMessage `json:"inputs"`
	ID      string          `json:"id"`
	Act     string          `json:"act"`
	Message string          `json:"message"`
}

// probeRoutes mounts the round's routes beside the kit:
//
//   - POST /probe/open files a Work Order of one of the plugin's own types
//     for the signed-in person, as that person (webui.As);
//   - POST /probe/say sends the person's turn to a Work Order, as that person;
//   - POST /probe/end interrupts, concludes or cancels a Work Order as the
//     plugin itself (webui.AsPlugin), and with act "deliver" tries to send a
//     turn as the plugin, which tera must refuse;
//   - GET /probe/calls lists what check_caller verified.
//
// Each answer is tera's reply verbatim: {"result": ...} or {"refusal": ...},
// or {"error": ...} when no verdict came back.
func probeRoutes(kit *webui.Kit, c start.Contract, calls *callLog) {
	kit.Handle("POST /probe/open", probeHandler(func(r *http.Request, in probeRequest) (workorder.Reply, error) {
		cl, err := webui.As(r)
		if err != nil {
			return workorder.Reply{}, err
		}
		typeKey, err := c.TypeKey(in.Type)
		if err != nil {
			return workorder.Reply{}, err
		}
		args := map[string]any{"type": typeKey, "worker": in.Worker, "title": in.Title}
		if len(in.Inputs) > 0 {
			args["inputs"] = in.Inputs
		}
		return invoke(r.Context(), cl, "work_order.create", args)
	}))
	kit.Handle("POST /probe/say", probeHandler(func(r *http.Request, in probeRequest) (workorder.Reply, error) {
		cl, err := webui.As(r)
		if err != nil {
			return workorder.Reply{}, err
		}
		return invoke(r.Context(), cl, "work_order.guide", map[string]any{"id": in.ID, "verb": "deliver", "message": in.Message})
	}))
	kit.Handle("POST /probe/end", probeHandler(func(r *http.Request, in probeRequest) (workorder.Reply, error) {
		cl, err := webui.AsPlugin(r)
		if err != nil {
			return workorder.Reply{}, err
		}
		switch in.Act {
		case "interrupt", "deliver":
			return invoke(r.Context(), cl, "work_order.guide", map[string]any{"id": in.ID, "verb": in.Act, "message": in.Message})
		case "conclude":
			return invoke(r.Context(), cl, "work_order.transition", map[string]any{"id": in.ID, "action": "conclude", "note": in.Message})
		case "cancel":
			return invoke(r.Context(), cl, "work_order.cancel", map[string]any{"id": in.ID, "reason": in.Message})
		}
		return workorder.Reply{}, errors.New(`act is one of "interrupt", "conclude", "cancel" and "deliver"`)
	}))
	kit.Handle("GET /probe/calls", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := pluginsdk.VerifiedCaller(r.Context()); !ok {
			http.Error(w, "sign in first", http.StatusUnauthorized)
			return
		}
		b, err := calls.read()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write(b)
	}))
}

func invoke(ctx context.Context, cl *webui.Clients, verb string, args map[string]any) (workorder.Reply, error) {
	params, err := json.Marshal(args)
	if err != nil {
		return workorder.Reply{}, err
	}
	return cl.Invoker.Invoke(ctx, verb, params)
}

func probeHandler(do func(*http.Request, probeRequest) (workorder.Reply, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in probeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "the body is one JSON object", http.StatusBadRequest)
			return
		}
		reply, err := do(r, in)
		out := map[string]any{}
		switch {
		case err != nil:
			out["error"] = err.Error()
		case len(reply.Refusal) > 0:
			out["refusal"] = reply.Refusal
		default:
			out["result"] = reply.Result
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}
