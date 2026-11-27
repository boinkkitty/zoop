import { describe, expect, it } from "vitest";
import { pendingSwitch, searchZones } from "./timezones";

describe("the pending timezone switch", () => {
  it("names the place and the Day it takes effect", () => {
    expect(
      pendingSwitch({
        pendingTimezone: "Asia/Tokyo",
        timezoneSwitchDay: "2026-11-26",
      }),
    ).toBe("Switching to Tokyo on Thu 26 Nov");
  });

  it("is nothing when no switch is pending", () => {
    expect(
      pendingSwitch({ pendingTimezone: null, timezoneSwitchDay: null }),
    ).toBeNull();
  });
});

describe("searching timezones", () => {
  const zones = [
    "America/Argentina/Buenos_Aires",
    "America/New_York",
    "Asia/Tokyo",
    "Europe/London",
  ];

  it("lists every zone before anything is typed", () => {
    expect(searchZones("  ", zones)).toEqual(zones);
  });

  it("finds a city as people write it, ignoring case", () => {
    expect(searchZones("new york", zones)).toEqual(["America/New_York"]);
    expect(searchZones("buenos", zones)).toEqual([
      "America/Argentina/Buenos_Aires",
    ]);
  });

  it("finds every zone in a region", () => {
    expect(searchZones("america", zones)).toEqual([
      "America/Argentina/Buenos_Aires",
      "America/New_York",
    ]);
  });

  it("finds nothing for a place it doesn't know", () => {
    expect(searchZones("Atlantis", zones)).toEqual([]);
  });
});
