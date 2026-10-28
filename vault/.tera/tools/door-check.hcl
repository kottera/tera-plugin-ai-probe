# door-check — the door-check type's session: it calls check_caller on the
# reserved tera server, then through the plugin's derived exposure with the
# dispatch's own token, and reports both answers verbatim.
tool "door-check" {
  description = "Calls check_caller on the reserved tera server and through the plugin's exposure, and reports both answers."

  runtime = "python3"
  entry   = "door_check.py"

  mcp_servers {
    server "ai-probe" {
      exposure = "plugin.ai-probe"
      tools    = ["check_caller"]

      credential {
        filer {}
      }
    }
  }

  sandbox {
    timeout = "120s"
  }
}
