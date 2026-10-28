// The app. It talks to tera only through the kit's routes on this page's own
// origin, with @kottera/tera-web's TeraClient: the browser never holds a tera
// credential, and every call acts as the person signed in. How it looks is
// tera's UI kit (index.html): this file builds markup from the kit's patterns
// and elements and writes no CSS. The person, the other apps and signing out
// are the shell's.
import { TeraClient, KitRequestError, type WorkOrderListResult } from "@kottera/tera-web";

// The app's base: "/" at the root of the plugin's own origin, "/plugins/<name>/"
// on a path-based installation. The build puts this module in <base>assets/
// (vite.config.ts), so its own URL names the base.
const base = new URL(/* @vite-ignore */ "../", import.meta.url).pathname;
const tera = new TeraClient({ base });

// One row of the list, as the generated type has it.
type WorkOrder = NonNullable<WorkOrderListResult extends { work_orders: infer W } ? W : never>[number];

function byId(id: string): HTMLElement {
  const el = document.getElementById(id);
  if (el === null) throw new Error("index.html has no #" + id);
  return el;
}

// el builds an element: a native one, or one of the UI kit's (tera-status,
// tera-side-panel, ...), whose attributes are its whole interface.
function el(tag: string, attrs: Record<string, string> = {}, ...kids: (Node | string)[]): HTMLElement {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  e.append(...kids);
  return e;
}

// The UI kit's state token for a Work Order state: violet waits on you, amber
// on the system, red is broken, green done, blue under way, the greys nobody
// needs to look at.
function token(state: string): string {
  if (state === "complete") return "done";
  if (state === "active") return "running";
  if (state === "draft") return "draft";
  if (state === "scheduled") return "idle";
  if (state === "failed") return "failed";
  if (state === "cancelled" || state === "superseded") return "cancelled";
  if (["paused:awaiting-input", "paused:awaiting-review", "paused:awaiting-credential", "failed:awaiting-ack"].includes(state)) {
    return "attention";
  }
  return "blocked";
}

// A type's own name, the last part of its key: claude-task, from
// plugin.claude-code.v1/work-order-types/claude-task. The panel shows the key.
function leaf(key: string): string {
  return key.slice(key.lastIndexOf("/") + 1);
}

// A state as a person reads it: "paused: awaiting input".
function label(state: string): string {
  return state.replace(":", ": ").replaceAll("-", " ");
}

// When something happened, in words, with the instant behind it.
function when(iso: string): HTMLElement {
  const at = new Date(iso);
  const s = Math.round((at.getTime() - Date.now()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const [n, unit]: [number, Intl.RelativeTimeFormatUnit] =
    Math.abs(s) < 60 ? [s, "second"] : Math.abs(s) < 3600 ? [Math.round(s / 60), "minute"] :
    Math.abs(s) < 86400 ? [Math.round(s / 3600), "hour"] : [Math.round(s / 86400), "day"];
  return el("time", { datetime: iso, title: at.toLocaleString() }, rtf.format(n, unit));
}

// The side panel: one at a time, beside the main column.
let panel: HTMLElement | null = null;

function closePanel(): void {
  const selected = byId("work").querySelector<HTMLElement>("[aria-current=true]");
  selected?.removeAttribute("aria-current");
  panel?.remove();
  panel = null;
  byId("page").removeAttribute("data-with-panel");
  selected?.focus();
}

function prop(term: string, value: Node | string): Node[] {
  return [el("dt", {}, term), el("dd", {}, value)];
}

// openPanel shows one Work Order in a <tera-side-panel>: its status first,
// then what it is. The element takes its children once, when it connects, so
// each row opens a new one.
function openPanel(wo: WorkOrder, row: HTMLElement): void {
  closePanel();
  row.setAttribute("aria-current", "true");
  const labels = wo.tags ?? [];
  panel = el("tera-side-panel", {
    heading: wo.title ?? wo.id, kind: "work-order", meta: leaf(wo.type),
    "status-token": token(wo.state), "status-label": label(wo.state), closable: "",
  },
    el("tera-section", { heading: "The Work Order" },
      el("dl", { class: "tera-props" },
        ...prop("ID", el("tera-identifier", { value: wo.id })),
        ...prop("Type", wo.type),
        ...prop("Worker", wo.worker || el("tera-empty", { kind: "none" })),
        ...prop("Filed", when(wo.created_at)),
        ...prop("Updated", when(wo.updated_at)),
      )),
    labels.length === 0
      ? el("tera-section", { heading: "Tags", aside: "None" })
      : el("tera-section", { heading: "Tags", aside: String(labels.length) },
        el("ul", { class: "tera-items" }, ...labels.map((t) => el("li", {}, el("span", { class: "tera-item-name" }, t))))),
  );
  panel.addEventListener("tera-close", closePanel);
  byId("page").setAttribute("data-with-panel", "");
  byId("page").append(panel);
}

function workRow(wo: WorkOrder): HTMLElement {
  const row = el("li", { "data-open": "", tabindex: "0" },
    el("div", {},
      el("div", { class: "tera-item-name" }, wo.title ?? wo.id),
      el("div", { class: "tera-item-meta" }, leaf(wo.type) + " · filed ", when(wo.created_at))),
    el("span", { class: "tera-item-end" }, el("tera-status", { token: token(wo.state), label: label(wo.state) })));
  row.addEventListener("click", () => openPanel(wo, row));
  row.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      openPanel(wo, row);
    }
  });
  return row;
}

// The Work Orders the person can read: work_order.list, invoked as them, with
// no filter, answers every Work Order their grants let them read — other
// people's too, for an auditor — not only the ones they filed. tera decides
// it on their grants, and on manifest.hcl's grant, which must name the verb.
async function showWork(): Promise<void> {
  const status = byId("work-status");
  const got = await tera.invoke<WorkOrderListResult>("work_order.list", { limit: 20 });
  if (got.status === "error") {
    // The verb ran and said no: its refusal, as tera gave it.
    status.replaceWith(el("div", { class: "tera-notice", "data-tone": "failed", id: "work-status" }, "tera refused: " + got.error.message));
    byId("work-count").textContent = "";
    return;
  }
  // work_orders may be null rather than an empty list: the schema says so,
  // and the generated type and strict mode hold the app to it.
  const workOrders = got.result.work_orders ?? [];
  byId("work").replaceChildren(...workOrders.map(workRow));
  status.hidden = true;
  byId("work-empty").hidden = workOrders.length > 0;
  byId("work-count").textContent = workOrders.length === 1 ? "1 Work Order" : workOrders.length + " Work Orders";
}

// The events watch.go has seen: a route of the plugin's own (events.go),
// under the app's base.
async function showEvents(): Promise<void> {
  const resp = await fetch(base + "events", { credentials: "same-origin", cache: "no-store" });
  if (!resp.ok) return;
  const { events } = (await resp.json()) as { events: string[] };
  byId("events").replaceChildren(...events.map((e) => el("li", {}, el("span", { class: "tera-item-name" }, e))));
  byId("events-empty").hidden = events.length > 0;
  byId("events-aside").textContent = events.length === 0 ? "" : String(events.length);
}

async function load(): Promise<void> {
  try {
    await Promise.all([showWork(), showEvents()]);
  } catch (err) {
    // The kit's own refusal: nobody signed in, a route not served, tera
    // unreachable. retryable says whether the same call may succeed later.
    const why = err instanceof KitRequestError ? err.code + ": " + err.message : String(err);
    byId("work-count").textContent = "";
    byId("work-status").replaceWith(el("div", { class: "tera-notice", "data-tone": "failed", id: "work-status" },
      "The UI could not reach tera (" + why + ")."));
  }
}

byId("refresh").addEventListener("click", () => void load());
void load();
