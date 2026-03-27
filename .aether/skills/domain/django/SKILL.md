---
name: django
description: Use when the project uses Django for Python web development
type: domain
domains: [backend, python, web]
agent_roles: [builder]
detect_files: ["manage.py"]
detect_packages: ["django"]
priority: normal
version: "1.0"
---

# Django Best Practices

## Project Structure

Follow Django's app-based architecture. Each app should represent a distinct feature or domain concept. Keep apps small and focused -- if an app has 20+ models, it probably needs splitting.

Place reusable utilities in a `core` or `common` app. Keep `settings.py` clean by splitting into `base.py`, `development.py`, and `production.py` using a settings package.

## Models

Define explicit `Meta` classes with `ordering`, `verbose_name`, and `db_table` where appropriate. Add `__str__` methods to every model for admin and debugging readability.

Use model managers for complex queries rather than scattering `filter()` chains through views. Create custom querysets with `as_manager()` for chainable query methods.

Always create migrations after model changes with `makemigrations`. Review generated migration files before running them. Never edit migrations manually unless you understand the implications.

## Views

Prefer class-based views (CBVs) for CRUD operations -- they eliminate boilerplate. Use function-based views for complex custom logic where CBV mixins would become contorted. In Django REST Framework, use `ModelViewSet` for standard CRUD and `APIView` for custom endpoints.

Keep views thin. Business logic belongs in services or model methods, not in views. A view should validate input, call a service, and return a response.

## Security

Never expose raw model instances to API responses -- use serializers or forms to control which fields are visible. Always validate and clean form input. Use Django's CSRF protection and never disable it.

Set `DEBUG = False` in production. Configure `ALLOWED_HOSTS` explicitly. Use environment variables for `SECRET_KEY` and database credentials.

## Database

Use `select_related()` for foreign key joins and `prefetch_related()` for reverse and many-to-many relationships to avoid N+1 query problems. Use Django Debug Toolbar in development to spot excessive queries.

Write data migrations for schema changes that need data transformation. Keep schema and data migrations separate.

## Testing

Use `TestCase` for tests that need database access, `SimpleTestCase` for those that do not. Use `factory_boy` or model `baker` over fixtures for test data. Test views through the test client, not by calling view functions directly.
