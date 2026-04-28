---
name: swift
description: Use when the project uses Swift for iOS, macOS, watchOS, tvOS, or visionOS development
type: domain
domains: [mobile, desktop, spatial-computing]
agent_roles: [builder]
detect_files: ["*.swift", "Package.swift", "*.xcodeproj", "*.xcworkspace", "project.pbxproj"]
detect_packages: ["swift", "swiftui", "uikit", "alamofire", "swiftdat"]
priority: normal
version: "1.0"
---

# Swift/SwiftUI Best Practices

## SwiftUI View Composition

- Break views into small, single-responsibility components; extract reusable pieces into `View` structs
- Use `@ViewBuilder` for conditional and grouped content; prefer composition over deep nesting
- Apply modifiers consistently -- chain them in a logical order: layout -> appearance -> accessibility
- Use `ViewModifier` protocol for reusable modifier groups shared across views
- Keep view bodies free of business logic; move computation to ViewModels or use computed properties on the model

## Swift Concurrency

- Use `async/await` as the primary concurrency model; avoid completion handlers in new code
- Adopt `Actor` for shared mutable state isolation: `actor DataStore { ... }`
- Use `TaskGroup` for structured concurrency when spawning dynamic child tasks
- Apply `.task` modifier in SwiftUI for view-scoped async work that cancels on disappear
- Use `Sendable` conformance to guarantee thread-safe value types across concurrency boundaries

## SwiftData

- Define models with `@Model` macro; use `@Relationship` for references and `@Attribute` for constraints
- Use `ModelContainer` to configure storage: in-memory for previews, persistent for production
- Query with `@Query` in SwiftUI views; apply `#Predicate` for type-safe filtering and sorting
- Keep model types simple -- derive computed properties for presentation logic rather than storing derived data
- Use `ModelContext` directly in ViewModels for batch inserts, deletes, and background saves

## UIKit Interop

- Use `UIViewControllerRepresentable` and `UIViewRepresentable` to wrap UIKit components in SwiftUI
- Delegate pattern: implement the delegate protocol in a coordinator class nested inside the representable
- Use `UIHostingController` to embed SwiftUI views within UIKit navigation flows
- Prefer SwiftUI-native solutions before falling back to UIKit; only interop when no SwiftUI equivalent exists

## visionOS Spatial Computing

- Use `RealityView` for 3D content and `ImmersiveSpace` for full-room experiences
- Apply `Volume` for bounded 3D windows and `Window` for traditional 2D-style panels
- Use ARKit skeleton tracking and hand tracking via `HandTrackingProvider` for spatial input
- Design for comfort: keep content at comfortable distances, avoid rapid depth changes, provide rest states
- Test spatial UI in the visionOS simulator with varied room layouts before device testing

## Testing and Quality

- Use Swift Testing framework (`@Test`, `@Suite`) for new test targets alongside XCTest for compatibility
- Write `PreviewProvider` or `#Preview` macros for every SwiftUI view to enable live canvas development
- Apply SwiftLint or SwiftFormat consistently; enforce rules in CI with `swift-format`
- Use Instruments for profiling: Time Profiler, Allocations, and Leaks for performance validation
