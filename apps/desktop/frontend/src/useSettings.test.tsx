import { act, renderHook, waitFor } from "@testing-library/react";
import { DEFAULT_SETTINGS } from "@tiles-spliter/shared";
import { describe, expect, it, vi } from "vitest";
import { setApiForTests } from "./api";
import { useSettings } from "./useSettings";

describe("useSettings", () => {
  it("loads settings then patches optimistically and persists", async () => {
    const update = vi.fn().mockResolvedValue(undefined);
    setApiForTests({
      getSettings: async () => structuredClone(DEFAULT_SETTINGS),
      updateSettings: update,
    });
    const { result } = renderHook(() => useSettings());
    await waitFor(() => expect(result.current.settings).not.toBeNull());
    act(() =>
      result.current.patch((s) => ({
        ...s,
        general: { ...s.general, windowPadding: 8 },
      })),
    );
    expect(result.current.settings?.general.windowPadding).toBe(8);
    await waitFor(() => expect(update).toHaveBeenCalledOnce());
    expect(update.mock.calls[0][0].general.windowPadding).toBe(8);
  });
});
