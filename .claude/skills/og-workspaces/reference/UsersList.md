# Reference: `UsersList`

The `UsersList` widget allows displaying users data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'users'
* `type`: Always must be 'UsersList'

### Paths in columns config

Columns path (._current.* not allowed) for UsersList:

| Field | Description |
|-------|-------------|
| email | User email address. |
| description | User description. |
| workgroup | Name of the workgroup the user belongs to. |
| domain | Name of the domain associated with the user. |
| profile | Name of the user profile. |
| name | User's first name. |
| surname | User's surname or last name. |
| countryCode | Country code associated with the user. |
| langCode | Language code associated with the user. |
| timezone | User's timezone. |
| loginWithPassword | Indicates whether the user can log in with a password. |


### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Fields for filter:

| Field | Description |
|-------|-------------|
| user.email | User email address. |
| user.description | User description. |
| workgroup.name | Name of the workgroup the user belongs to. |
| domain.name | Name of the domain associated with the user. |
| profile.name | Name of the user profile. |
| user.name | User's first name. |
| user.surname | User's surname or last name. |
| country.code | Country code associated with the user. |
| language.code | Language code associated with the user. |
| user.timezone | User's timezone. |
| loginWithPassword | Indicates whether the user can log in with a password. |
