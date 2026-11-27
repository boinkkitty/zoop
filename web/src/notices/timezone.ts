// The timezone Notice: raised on load when the browser's zone isn't the one the
// user's Days are on, or about to be. Keep is remembered per detected zone here.

type Profile = { timezone: string; pendingTimezone: string | null };

export type TimezoneNotice = {
  zone: string; // the browser's zone, sent by Switch
  place: string; // "Tokyo"
  currentPlace: string; // where the Days are, or are headed: "Keep London"
};

const keptKey = (zone: string) => `zoop.timezone.kept.${zone}`;

export function timezoneNotice({
  browserZone,
  me,
  storage,
}: {
  browserZone: string;
  me: Profile;
  storage: Storage;
}): TimezoneNotice | null {
  const headedTo = me.pendingTimezone ?? me.timezone;
  if (browserZone === headedTo || kept(storage, browserZone)) return null;
  return {
    zone: browserZone,
    place: placeName(browserZone),
    currentPlace: placeName(headedTo),
  };
}

// keepTimezone stops asking about zone in this browser.
export function keepTimezone(storage: Storage, zone: string) {
  try {
    storage.setItem(keptKey(zone), "1");
  } catch {
    // Storage full or blocked: Keep still hides the Notice until the next load.
  }
}

function kept(storage: Storage, zone: string) {
  try {
    return storage.getItem(keptKey(zone)) !== null;
  } catch {
    return false;
  }
}

// placeName is the city of an IANA zone: "America/Argentina/Buenos_Aires" → "Buenos Aires".
const placeName = (zone: string) =>
  (zone.split("/").pop() ?? zone).replaceAll("_", " ");
