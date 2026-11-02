plugin_manifest "ai-probe" {
  version = "0.2.1"

  requires {
    # 20: adoption {} below (ADR-193 D5).
    level "manifest" { min = 20 }
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

  adoption {
    rules = <<-EOT
      The AI probe: a UI that opens an `assistant`, a live Claude Code
      conversation, for the person signed in to it, sends that assistant the
      person's turns and ends it, and files a `door-check` that calls the
      plugin's served tool `check_caller` through both of tera's doors and
      reports the two answers. `check_caller` answers with the caller assertion
      tera attached to the call, as the plugin verified it, and the plugin keeps
      each answer in its own data directory. It is a probe of how a plugin acts
      for the people it serves: it does work for nobody else, and nothing runs it
      unwatched.

      ## The grants it asks for

      Both are requested by the manifest and granted only by the installation's
      approval. Approve them to use the plugin; without them its UI opens nothing
      and ends nothing.

      - **`converse-for-a-person`** lets the plugin, as the signed-in person,
        file an `assistant` or a `door-check` (`work_order/create`) and send an
        assistant that person's turns (`work_order/guide`). It reaches only the
        plugin's own two types. The Work Order records the person as its creator,
        so the person must be one who may file those types.
      - **`end-assistants`** lets the plugin, as itself, interrupt, conclude and
        cancel the `assistant` Work Orders it opened. It sends no turn as
        itself: a turn is always a person's, under `converse-for-a-person`.

      ## What an adoption holds for the types to work

      - **The plugin it builds on, installed:** the `claude-code` harness, which
        runs the `assistant`'s session, and whatever that harness itself builds
        on to reach the model.
      - **A worker that runs `claude-code` with a model credential.** The
        `assistant` type names no credential; the deployment's binding for the
        harness supplies one, as it does for any type of that harness.
      - **A `python3` on the worker that runs `door-check`.** Its tool is a
        Python script, and the plugin builds on no toolchain that provides one.
      - **Nothing in the repository.** Neither type clones a repository, reads a
        file of one, or asks anything of a repository's process.

      ## The model

      The `assistant` type names no model, no tier and no effort, so its session
      runs whatever the `claude-code` installation resolves for a type that names
      none. The person chooses the model, through that installation's settings;
      this plugin sets none. Write no model into the repository for it.

      ## The skills

      This plugin's vault carries no skill, and its sessions read none.
    EOT
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
