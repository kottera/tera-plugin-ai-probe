#!/usr/bin/env python3
# Calls check_caller on the reserved tera server (tera-mcp-tera) and through
# the plugin's exposure (tera-mcp-ai-probe), and reports both answers.
import json, os, select, shutil, struct, subprocess, sys

sys.stdin.read()
report = {"error": ""}

def session(server):
    exe = shutil.which("tera-mcp-" + server)
    if not exe:
        raise RuntimeError("tera-mcp-%s is not on PATH" % server)
    p = subprocess.Popen([exe], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    state = {"id": 0}
    def send(msg):
        p.stdin.write((json.dumps(msg) + "\n").encode())
        p.stdin.flush()
    def request(method, params):
        state["id"] += 1
        send({"jsonrpc": "2.0", "id": state["id"], "method": method, "params": params})
        while True:
            ready, _, _ = select.select([p.stdout], [], [], 60)
            if not ready:
                raise RuntimeError("no answer from tera-mcp-%s within 60s" % server)
            line = p.stdout.readline()
            if not line:
                raise RuntimeError("tera-mcp-%s closed its stdout" % server)
            msg = json.loads(line)
            if msg.get("id") == state["id"]:
                return msg
    init = request("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                                  "clientInfo": {"name": "door-check", "version": "1"}})
    if "error" in init:
        raise RuntimeError("initialize on %s: %s" % (server, init["error"]))
    send({"jsonrpc": "2.0", "method": "notifications/initialized"})
    return p, request

def close(p):
    try:
        p.stdin.close()
        p.wait(timeout=10)
    except Exception:
        p.kill()

try:
    p, request = session("tera")
    try:
        listed = request("tools/list", {})
        report["reserved_tools"] = [t["name"] for t in listed.get("result", {}).get("tools", [])]
        report["reserved"] = request("tools/call", {"name": "check_caller", "arguments": {"note": "door-check: reserved"}})
    finally:
        close(p)
    p, request = session("ai-probe")
    try:
        report["gateway"] = request("tools/call", {"name": "check_caller", "arguments": {"note": "door-check: gateway"}})
    finally:
        close(p)
except Exception as e:
    report["error"] = str(e)

decl = {"status": "complete", "summary": json.dumps(report)}
payload = json.dumps({"type": "result", "status": "ok", "structured": decl}).encode()
os.write(3, struct.pack(">I", len(payload)))
os.write(3, payload)
