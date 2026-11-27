// The Settings timezone row: a searchable list of IANA zones, switching from tomorrow.
import { placeName } from "../notices/timezone";

type Profile = {
  pendingTimezone: string | null;
  timezoneSwitchDay: string | null;
};

const dayLabel = new Intl.DateTimeFormat("en-GB", {
  weekday: "short",
  day: "numeric",
  month: "short",
  timeZone: "UTC",
});

// pendingSwitch says where the Days are headed and from which Day: "Switching to Tokyo on Thu 26 Nov".
export function pendingSwitch(me: Profile): string | null {
  if (!me.pendingTimezone || !me.timezoneSwitchDay) return null;
  const day = dayLabel.format(new Date(`${me.timezoneSwitchDay}T00:00:00Z`));
  return `Switching to ${placeName(me.pendingTimezone)} on ${day.replace(",", "")}`;
}

// searchZones keeps the zones whose name holds query, with "_" read as a space: "new york" finds America/New_York.
export function searchZones(query: string, zones: readonly string[]): string[] {
  const q = query.trim().toLowerCase();
  if (!q) return [...zones];
  return zones.filter((z) => z.replaceAll("_", " ").toLowerCase().includes(q));
}
