import { act, renderHook } from "@testing-library/react";
import { DEFAULT_SETTINGS } from "@tiles-spliter/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setApiForTests } from "./api";
import { useSettings } from "./useSettings";

// Flush pending microtasks (promise .then chains) inside act() so React
// state updates from resolved promises are applied and asserted safely.
async function flushMicrotasks(times = 3) {
  for (let i = 0; i < times; i++) {
    // eslint-disable-next-line no-await-in-loop
    await act(async () => {
      await Promise.resolve();
    });
  }
}

describe("useSettings", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("loads settings, patches optimistically, then debounces a single persist", async () => {
    const update = vi.fn().mockResolvedValue(undefined);
    setApiForTests({
      getSettings: async () => structuredClone(DEFAULT_SETTINGS),
      updateSettings: update,
    });
    const { result } = renderHook(() => useSettings());
    await flushMicrotasks();
    expect(result.current.settings).not.toBeNull();

    act(() => {
      result.current.patch((s) => ({
        ...s,
        general: { ...s.general, windowPadding: 4 },
      }));
    });
    act(() => {
      result.current.patch((s) => ({
        ...s,
        general: { ...s.general, windowPadding: 8 },
      }));
    });

    // Local state updates immediately (optimistic); persistence is debounced.
    expect(result.current.settings?.general.windowPadding).toBe(8);
    expect(update).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200);
    });
    await flushMicrotasks();

    // Two rapid patches within the debounce window collapse into one call
    // carrying only the final, settled value.
    expect(update).toHaveBeenCalledOnce();
    expect(update.mock.calls[0][0].general.windowPadding).toBe(8);
  });

  it("falls back to DEFAULT_SETTINGS so the UI still renders when the load fails", async () => {
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    setApiForTests({
      getSettings: async () => {
        throw new Error("bindings unavailable");
      },
    });
    const { result } = renderHook(() => useSettings());
    await flushMicrotasks();

    expect(result.current.settings).toEqual(DEFAULT_SETTINGS);
    expect(errSpy).toHaveBeenCalled();
    errSpy.mockRestore();
  });

  it("a failed persist does not poison the chain — a later edit still persists", async () => {
    const update = vi
      .fn()
      .mockRejectedValueOnce(new Error("network down"))
      .mockResolvedValueOnce(undefined);
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    setApiForTests({
      getSettings: async () => structuredClone(DEFAULT_SETTINGS),
      updateSettings: update,
    });
    const { result } = renderHook(() => useSettings());
    await flushMicrotasks();

    act(() => {
      result.current.patch((s) => ({
        ...s,
        general: { ...s.general, windowPadding: 4 },
      }));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(200);
    });
    await flushMicrotasks();

    expect(update).toHaveBeenCalledTimes(1);

    // A second, later edit must still persist even though the first write
    // rejected — a poisoned chain would silently skip every .then() after
    // the rejection and this call would never fire.
    act(() => {
      result.current.patch((s) => ({
        ...s,
        general: { ...s.general, windowPadding: 8 },
      }));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(200);
    });
    await flushMicrotasks();

    expect(update).toHaveBeenCalledTimes(2);
    expect(update.mock.calls[1][0].general.windowPadding).toBe(8);
    errSpy.mockRestore();
  });

  it("replaceLocal updates state without persisting", async () => {
    const update = vi.fn().mockResolvedValue(undefined);
    setApiForTests({
      getSettings: async () => structuredClone(DEFAULT_SETTINGS),
      updateSettings: update,
    });
    const { result } = renderHook(() => useSettings());
    await flushMicrotasks();

    act(() => {
      result.current.replaceLocal((s) => ({
        ...s,
        hotkeys: { ...s.hotkeys, enabled: false },
      }));
    });

    expect(result.current.settings?.hotkeys.enabled).toBe(false);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200);
    });
    await flushMicrotasks();
    expect(update).not.toHaveBeenCalled();
  });
});
