# door-check — one dispatch that calls check_caller on both doors: the
# reserved tera server and the plugin's derived exposure through the gateway.
# The two assertions must agree on every claim but iat, exp and jti.
work_order_type "door-check" {
  description = "Calls the ai-probe plugin's check_caller once on the reserved tera server and once through the gateway, in one dispatch, and reports both answers."

  capability "plugin.ai-probe/tools/door-check" {
    operation = "invoke"
  }

  in_dispatch {
    verbs = []
    tool "plugin.ai-probe/served-tools/check_caller" {}
  }

  defaults {
    max_retries = 0
  }

  postconditions {
    none = "The report is read off the Work Order's outputs and the plugin's record, and compared by the person running the round."
  }
}
