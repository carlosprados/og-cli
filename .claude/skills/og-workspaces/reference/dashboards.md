# Dashboard Configurations

Dashboards define the global view configurations (`_id`, `title`, `icon`, etc.). 

## Dashboard JSON Structure

Here it is a full example of a dashboard JSON configuration:

```json
{
  "extraConfig": {
    "cellsWidth": "",
    "cellHeight": 145,
    "dashboardRefreshInterval": "",
    "showBanner": false,
    "favourite": false
  },
  "_id": "3ab97a11-863b-4f79-bfec-95852a1aa268",
  "icon": "fa-tachometer",
  "iconType": "icon",
  "backgroundImageSize": "100% 100%",
  "users": [
    
  ],
  "workspaces": "_1696931515204_342",
  "workgroups": [
    
  ],
  "allowedProfiles": [
    "root",
    "super_admin_domain",
    "admin_domain",
    "admin",
    "advanced",
    "viewer"
  ],
  "domains": [
    
  ],
  "lastAccess": "2026-05-22T07:06:29.436Z",
  "id": "3ab97a11-863b-4f79-bfec-95852a1aa268",
  "owner": "josemaria.perez@amplia.es",
  "title": "listado entidades",
  "description": null,
  "editable": true,
  "backgroundImage": null,
  "backgroundColor": "",
  "grid": [],
  "templateConfig": null
}
```

## Dashboard metadata

| Field | Value | Data Type | Required | Description / Details |
| :--- | :--- | :--- | :--- | :--- |
| **_id** | `dashboard-id` | String | Yes  | Dashboard identifier |
| **id** | `dashboard-id` | String | Yes  | Dashboard identifier |
| **workspaces** | `_1696931515204_342` | String | Yes  | Workspace identifier |
| **title** | `dash title` | String | No  | Dashboard title |
| **description** | `dash description` | String | No  | Dashboard description |
| **owner** | `username` | String | Yes  | Username of the dashboard owner |
| **icon** | `fa-tachometer` | String | Yes  | Assigned icon (FontAwesome 4 class name) if `iconType` is 'icon', otherwise it is a URL |
| **iconType** | `icon` | String | Yes  | Icon type (Only values: 'icon' or 'image') |
| **allowedProfiles** | `["root", "super_admin_domain", "admin_domain", "admin", "advanced", "viewer"]` | Array | Yes  | Array of profiles that have access to the dashboard. Always have this value set to this array. |
| **users** | `[]` | Array | Yes  | Array of users that have readable access to the dashboard |
| **workgroups** | `[]` | Array | Yes  | Unused |
| **domains** | `[]` | Array | Yes  | Array of organizations that have readable access to the dashboard |
| **lastAccess** | `2026-05-22T07:06:29.436Z` | String | No  | Last access date to the dashboard | |
| **editable** | `true` | Booleano | Yes  | Always true |
| **backgroundImageSize** | `100% 100%` | String | Yes  | Background image size. Values: '100% 100%'. Deprecated |
| **backgroundImage** | `` | String (URL) | No  | Background image URL to show in workspace |
| **bannerImage** | `` | String (URL) | No  | Banner image URL to show in dashboard |
| **templateConfig** | *null* | Null | No | template export configuration |
| **backgroundColor** | *null* | Null | No | Deprecated|
| **grid** | `[]` | Array | No  | Widget grid items |
| **extraConfig.cellsWidth** | `-small` | String | No | Grid cells width. Values: '' (default, 12 columns), '-small' (24 columns) |
| **extraConfig.cellHeight** | `50` | Integer | No  | Grid cells height (minimum value is 10) |
| **extraConfig.dashboardRefreshInterval** | `1800` | String (Number) | No | Dashboard refresh interval in seconds (Values: 0, 5(minutes), 10(minutes), 15(minutes), 30(minutes), 1(hour)), always in seconds format |
| **extraConfig.showBanner** | `false` | Booleano | No  | Shows the banner (header) of the dashboard |
| **extraConfig.favourite** | `false` | Booleano | No  | Mark if the dashboard is a favorite |
# Widgets Included in Example Dashboard

The example dashboard includes the following widgets:

- **OGScheduleHistoryBrowser** – Schedules history list widget.
- **rulesBrowser** – Read‑only view of platform rules. See [Reference: Rules Browser](./rulesBrowser.md).
- **customTable** – Custom table widget for plant metrics. See [Reference: customTable](./customTable.md).
