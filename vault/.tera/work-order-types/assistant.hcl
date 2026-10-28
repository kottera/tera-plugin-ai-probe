# assistant — an AI the probe runs for one person: an interactive Claude Code
# session, granted the probe's own served tool check_caller on the reserved
# tera server. The probe files it for the signed-in person, sends it their
# turns, and ends it.
work_order_type "assistant" {
  description = "A conversation the ai-probe plugin opens for the signed-in person. Each turn may call check_caller, which answers with the caller assertion the plugin received."

  input_schema = <<EOT
{
  "type": "object",
  "required": ["task"],
  "properties": {
    "task": { "type": "string", "minLength": 1, "description": "The opening message." }
  },
  "additionalProperties": false
}
EOT

  capability "plugin.claude-code/tools/claude-code" {
    operation   = "invoke"
    interactive = true

    config {
      instructions = <<EOT
You are the ai-probe plugin's assistant, in a conversation with the person who
opened it. When a message asks you to check the caller, call the tool
mcp__tera__check_caller with the note the message gives, and answer with the
JSON it returned, verbatim. Otherwise do what the message asks.
EOT
    }
  }

  in_dispatch {
    verbs = []
    tool "plugin.ai-probe/served-tools/check_caller" {}
  }

  defaults {
    max_retries = 0
  }

  postconditions {
    none = "A conversation makes no claim to verify: the plugin ends it, and what it said is read off the Work Order and the plugin's own record."
  }
}
