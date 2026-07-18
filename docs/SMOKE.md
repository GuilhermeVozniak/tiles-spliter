# Release Smoke Checklist

Manual parity checklist run before every release. Check each item against a
real reference app (e.g. Safari) as the target window.

1. All 17 tray-menu actions place the window exactly (halves, quarters, thirds cycle L→M→R with wrap, two-thirds ×3, center keeps size, fullscreen fills visible frame).
2. Every default hotkey from the spec table fires its action; Enable Hotkeys off kills all of them; remap + restore defaults works.
3. Drag to each of the 8 zones: overlay previews after the configured delay, drop applies, disabled zone does nothing.
4. ⌥-drag and ⌘-drag to left/right edges snap to thirds.
5. Restore previous size: snap by drag, drag away → original size returns.
6. Undo (⌥⌘Y and menu) restores the pre-action frame repeatedly (stack).
7. Padding 10pt visibly gaps snapped windows; fullscreen padding only when enabled.
8. Two displays: next/prev display moves proportionally; snapping works on the second display; zones on both.
9. Animations toggle changes snap movement between animated/instant.
10. Launch at login registers (System Settings → Login Items); menu-bar icon hide → app keeps working, relaunch from Finder opens Preferences.
11. Kill the settings file with garbage → app launches with defaults, `.bak` created.
12. Non-resizable window (e.g. System Settings) → actions fail silently, no crash.
