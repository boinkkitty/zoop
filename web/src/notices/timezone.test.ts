import { beforeEach, describe, expect, it } from "vitest";
import { keepTimezone, timezoneNotice } from "./timezone";

function memoryStorage(): Storage {
  const m = new Map<string, string>();
  return {
    get length() {
      return m.size;
    },
    clear: () => m.clear(),
    getItem: (k) => m.get(k) ?? null,
    key: (i) => [...m.keys()][i] ?? null,
    removeItem: (k) => void m.delete(k),
    setItem: (k, v) => void m.set(k, v),
  };
}

const london = { timezone: "Europe/London", pendingTimezone: null };
let storage: Storage;
beforeEach(() => {
  storage = memoryStorage();
});

describe("the timezone Notice", () => {
  it("stays away while the browser is on the profile's zone", () => {
    expect(
      timezoneNotice({ browserZone: "Europe/London", me: london, storage }),
    ).toBeNull();
  });

  it("asks about the browser's zone, naming both places", () => {
    expect(
      timezoneNotice({ browserZone: "Asia/Tokyo", me: london, storage }),
    ).toEqual({ zone: "Asia/Tokyo", place: "Tokyo", currentPlace: "London" });
    expect(
      timezoneNotice({
        browserZone: "America/Argentina/Buenos_Aires",
        me: london,
        storage,
      })?.place,
    ).toBe("Buenos Aires");
  });

  it("stays away once a switch to the browser's zone is pending", () => {
    expect(
      timezoneNotice({
        browserZone: "Asia/Tokyo",
        me: { timezone: "Europe/London", pendingTimezone: "Asia/Tokyo" },
        storage,
      }),
    ).toBeNull();
  });

  it("asks about staying put when a switch elsewhere is pending, offering to keep it", () => {
    expect(
      timezoneNotice({
        browserZone: "Europe/London",
        me: { timezone: "Asia/Tokyo", pendingTimezone: "Europe/London" },
        storage,
      }),
    ).toBeNull();
    expect(
      timezoneNotice({
        browserZone: "Asia/Tokyo",
        me: { timezone: "Asia/Tokyo", pendingTimezone: "Europe/London" },
        storage,
      }),
    ).toEqual({ zone: "Asia/Tokyo", place: "Tokyo", currentPlace: "London" });
  });

  it("remembers Keep for that zone in this browser, and asks about another", () => {
    keepTimezone(storage, "Asia/Tokyo");
    expect(
      timezoneNotice({ browserZone: "Asia/Tokyo", me: london, storage }),
    ).toBeNull();
    expect(
      timezoneNotice({ browserZone: "Asia/Seoul", me: london, storage })?.zone,
    ).toBe("Asia/Seoul");
  });

  it("still asks when this browser won't store anything", () => {
    const blocked = {
      ...memoryStorage(),
      getItem: () => {
        throw new Error("SecurityError");
      },
      setItem: () => {
        throw new Error("QuotaExceededError");
      },
    };
    expect(() => keepTimezone(blocked, "Asia/Tokyo")).not.toThrow();
    expect(
      timezoneNotice({
        browserZone: "Asia/Tokyo",
        me: london,
        storage: blocked,
      })?.zone,
    ).toBe("Asia/Tokyo");
  });
});
