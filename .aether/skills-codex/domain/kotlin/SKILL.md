---
name: kotlin
description: Use when the project uses Kotlin for Android development, backend services, or Kotlin Multiplatform
type: domain
domains: [mobile, backend, cross-platform]
agent_roles: [builder]
detect_files: ["*.kt", "*.kts", "build.gradle.kts", "settings.gradle.kts"]
detect_packages: ["kotlin", "jetpack-compose", "coroutines", "koin", "ktor"]
priority: normal
version: "1.0"
---

# Kotlin/Android Best Practices

## Jetpack Compose

- Build UI with composable functions; keep each composable small, stateless, and reusable
- Hoist state to the caller: accept `(value: T, onChange: (T) -> Unit)` parameters instead of internal `remember`
- Use `ViewModel` with `StateFlow` or `Compose StateFlow` (`collectAsStateWithLifecycle`) for screen-level state
- Apply `MaterialTheme` and typography systems consistently; define custom theme extensions in a `theme` package
- Use `LazyColumn`/`LazyRow` with `key` for efficient list rendering; avoid nested scrolling conflicts

## Coroutines and Flow

- Use `viewModelScope.launch` for ViewModel-driven async work; it cancels automatically on ViewModel clearance
- Prefer `StateFlow` for hot state streams and `SharedFlow` for one-to-many events (snackbars, navigation)
- Apply `flowOn(Dispatchers.IO)` for upstream work; collect on `Dispatchers.Main` in Android
- Use `suspend` functions for single async operations; use `Flow` for streams of values
- Handle cancellation cooperatively: use `ensureActive()` or `yield()` in long-running loops

## KMP Shared Modules

- Structure multiplatform projects with `shared`, `androidApp`, and `iosApp` modules
- Use `expect`/`actual` for platform-specific implementations; keep the actual implementations minimal
- Share ViewModels, repositories, and domain logic across platforms; keep UI layer native per platform
- Use `kotlinx.serialization` for JSON handling across all targets instead of platform-specific libraries
- Test shared code in the `commonTest` source set using `kotlin-test` or `kotlinx-coroutines-test`

## Android Architecture Components

- Follow the data layer -> domain layer -> UI layer pattern; use `Repository` as the single source of truth
- Use `Room` for local persistence with `@Entity`, `@Dao`, and `@Database` annotations
- Apply `Hilt` or `Koin` for dependency injection; annotate ViewModels with `@HiltViewModel`
- Use `WorkManager` for guaranteed background execution; `ForegroundService` only when user-visible
- Handle lifecycle with `Lifecycle-aware components`; use `repeatOnLifecycle(STARTED)` for Flow collection

## Testing

- Write unit tests with `kotlinx-coroutines-test` using `runTest`; use `TestDispatcher` to control timing
- Use Turbine for Flow testing: `testFlow.test { assertEquals(expected, awaitItem()) }`
- Instrumentation tests with `androidx.test` for UI: use `compose:testRule` for Compose testing
- Apply `given/when/then` naming convention; mock with MockK for Kotlin-native mocking
