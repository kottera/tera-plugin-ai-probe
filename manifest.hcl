plugin_manifest "ai-probe" {
  version = "0.2.0"

  requires {
    level "manifest" { min = 17 }
    level "recipes" { min = 37 }
    negotiated "mcp" { versions = ["2025-11-25", "2025-06-18", "2025-03-26"] }
  }

  about {
    description = "The plugin-ai-authority dogfood round's probe: an AI a plugin runs for the people it serves."
    license     = "Apache-2.0"
    publisher   = "Kottera"
  }

  serves {
    tools {}
    ui {
      title   = "AI probe"
      summary = "Opens an assistant for you, sends it your turns, and ends it"
    }
  }

  # The assistant's session is a Claude Code session.
  builds_on "claude-code" { major_version = "v1" }

  grant "converse-for-a-person" {
    purpose = "Opens an assistant or a door check for the signed-in person and sends the assistant their turns, as that person."
    on_behalf_of {
      verbs = ["work_order/create", "work_order/guide"]
      types = [
        "plugin.ai-probe/work-order-types/assistant",
        "plugin.ai-probe/work-order-types/door-check",
      ]
    }
  }

  grant "end-assistants" {
    purpose = "Interrupts, concludes and cancels the assistants it opened for a person, as itself."
    ends {
      types = ["assistant"]
    }
  }
}
