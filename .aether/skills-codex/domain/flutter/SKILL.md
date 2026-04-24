---
name: flutter
description: Use when the project uses Flutter/Dart for cross-platform mobile, web, or desktop applications
type: domain
domains: [mobile, web, desktop, cross-platform]
agent_roles: [builder]
detect_files: ["pubspec.yaml", "lib/main.dart", "*.dart", "analysis_options.yaml"]
detect_packages: ["flutter", "dart", "riverpod", "flutter_bloc", "get"]
priority: normal
version: "1.0"
---

# Flutter/Dart Best Practices

## Widget Composition

- Build UI from small, composable widgets; each widget should do one thing well
- Use `const` constructors wherever possible -- enables compile-time optimizations and faster rebuilds
- Prefer stateless widgets; only use `StatefulWidget` when the widget manages mutable state
- Apply the `Widget` composition pattern: container widgets handle state, leaf widgets are purely presentational
- Extract repeated UI patterns into reusable widgets in a `widgets/` or `components/` directory

## State Management

- Use Riverpod as the primary state management solution: `Provider` for dependencies, `Notifier` for state
- Define providers at the top level; use `ref.watch` in widgets and `ref.read` in callbacks
- Use `AsyncValue` for loading/error/data state handling: `when(loading:, error:, data:)`
- For BLoC pattern: define events, states, and blocs; use `BlocBuilder`, `BlocListener`, `BlocConsumer` in UI
- Keep state immutable; create new state instances on change rather than mutating in place

## Platform Channels

- Use `MethodChannel` for calling platform-specific APIs from Dart to native (Kotlin/Swift)
- Use `EventChannel` for streaming platform events to Dart: sensor data, connectivity changes
- Register channels in `MainActivity` (Android) and `AppDelegate` (iOS) with matching channel names
- Use `Pigeon` for type-safe platform channel code generation instead of manual serialization
- Keep platform-specific code minimal; push as much logic as possible into the shared Dart layer

## Responsive Layout

- Use `MediaQuery` for screen-size-dependent layouts; `LayoutBuilder` for parent-constrained layouts
- Apply `Flexible` and `Expanded` within `Row`/`Column` for proportional sizing
- Use `SingleChildScrollView` with `ConstrainedBox` for scrollable forms; avoid unbounded heights
- Define breakpoints in a constants file: mobile (<600), tablet (600-1200), desktop (>1200)
- Test layouts with `DevicePreview` package across screen sizes before physical device testing

## Testing

- Write widget tests with `testWidgets`: pump the widget, find elements, verify behavior
- Use `mockito` or `mocktail` for mocking dependencies in unit tests
- Create golden tests for critical UI components: `matchesGoldenFile('goldens/widget.png')`
- Run `flutter test --coverage` in CI; enforce minimum coverage thresholds for core logic
- Integration tests in `integration_test/` use `IntegrationTestWidgetsFlutterBinding` for device testing
