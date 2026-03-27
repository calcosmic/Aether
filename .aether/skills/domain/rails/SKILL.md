---
name: rails
description: Use when the project uses Ruby on Rails for web development
type: domain
domains: [backend, ruby, web]
agent_roles: [builder]
detect_files: ["Gemfile", "config/routes.rb"]
priority: normal
version: "1.0"
---

# Ruby on Rails Best Practices

## Convention Over Configuration

Follow Rails conventions strictly. Name models singular (`User`), controllers plural (`UsersController`), tables plural (`users`). Place files where Rails expects them. Fighting conventions creates maintenance headaches.

Use Rails generators for scaffolding, then customize. They set up the correct file structure, naming, and test stubs.

## Models

Keep models focused. Move business logic into service objects or concerns when a model exceeds 200 lines. Use Active Record validations for data integrity and callbacks sparingly -- callbacks that trigger side effects (emails, API calls) are better handled in service objects.

Use scopes for reusable query patterns: `scope :active, -> { where(active: true) }`. Chain scopes for composable queries.

## Controllers

Keep controllers thin. A controller action should: authenticate, authorize, call a service, and render a response. Complex logic belongs in service objects (`app/services/`), not controllers.

Use strong parameters (`params.require(:user).permit(:name, :email)`) for every create/update action. Never trust user input.

## Database

Write migrations for every schema change -- never modify the database manually. Use `change` methods in migrations for reversibility. Add database-level constraints (NOT NULL, unique indexes, foreign keys) alongside Active Record validations.

Watch for N+1 queries. Use `includes()` for eager loading associations. The `bullet` gem detects N+1 queries in development.

## Testing

Write request specs over controller specs -- they test the full stack including routing and middleware. Use `FactoryBot` for test data, not fixtures (fixtures become stale and hard to maintain).

Test the happy path and the most important failure modes. Use `let` and `before` blocks in RSpec for DRY test setup, but do not over-abstract -- tests should be readable as documentation.

## Security

Never store secrets in source code. Use Rails encrypted credentials (`rails credentials:edit`) or environment variables. Keep `secret_key_base` secure -- it signs session cookies.

Sanitize user-generated HTML output. Rails escapes by default in ERB, but `raw` and `html_safe` bypass this -- use them only with trusted content.

## Performance

Cache aggressively with fragment caching and Russian doll caching. Use background jobs (Sidekiq, GoodJob) for anything that takes more than a few hundred milliseconds -- email delivery, API calls, report generation.
