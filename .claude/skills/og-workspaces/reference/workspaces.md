# Dashboard Configurations

Dashboards define the global view configurations (`_id`, `title`, `icon`, etc.). 

## Dashboard JSON Structure

Here it is a full example of a dashboard JSON configuration:

```json
{
    "others": {
        "bannerImage": "https://picsum.photos/200/300",
        "showInHome": true,
        "mode": "grid"
    },
    "_id": "_1696931515204_342",
    "users": [],
    "domains": [],
    "workgroups": [],
    "actions": [
        "editConnectorFunction",
        "manageManufacturers",
        "manageManufacturerModels",
        "createArea",
        "createAsset",
        "createBundle",
        "createCertificateCard",
        "createChannel",
        "createDataset",
        "createDevice",
        "executeOperation",
        "createDomain",
        "managePipelines",
        "createProvisionFunctions",
        "createSubscriber",
        "createSubscription",
        "createTicket",
        "createTimeseries",
        "createUser",
        "createWorkgroup",
        "createDatamodelCard",
        "deviceCollection",
        "manageOperationType",
        "wizardViews",
        "editRuleConfiguration",
        "launchBulk",
        "launchBulkProcessors",
        "manageTransformers",
        "manageAIModels",
        "manageDevicePlans",
        "manageOrgSoftware",
        "createRestRequest",
        "manageOrganizationPlans",
        "manageOrgManufacturerModels",
        "createImageExecution",
        "manageOrgManufacturers",
        "createTimeseriesFunctions",
        "createScheduler"
    ],
    "widgets": [],
    "allowedProfiles": [
        "root",
        "super_admin_domain",
        "admin_domain",
        "admin",
        "advanced",
        "viewer",
        "widget_action"
    ],
    "dashboards": [],
    "lastAccess": "2026-05-27T09:04:20.625Z",
    "owner": "josemaria.perez@amplia.es",
    "name": "Workspace title",
    "icon": "ogicon-abacus",
    "color": "#FF0000FF",
    "widget_action": {
        "deviceCollection": "deviceCollection",
        "editConnectorFunction": "connectorFunctions",
        "editConnectorFunctionsCatalog": "connectorFunctionsCatalog",
        "editRuleConfiguration": "ruleConfigurations",
        "editDevice": "entities",
        "editAsset": "assets",
        "createDevice": "entities",
        "createAsset": "assets",
        "createTicket": "tickets",
        "editTicket": "tickets",
        "createSubscription": "subscriptions",
        "editSubscription": "subscriptions",
        "createSubscriber": "subscribers",
        "editSubscriber": "subscribers",
        "launchBulk": "bulk",
        "launchBulkProcessors": "bulkProcessors",
        "createProvisionFunctions": "provisionFunctions",
        "createTimeseriesFunctions": "timeseriesFunction",
        "createChannel": "channels",
        "createUser": "user",
        "editUserWizard": "user",
        "createDomain": "domain",
        "editBundle": "bundles",
        "createBundle": "bundles",
        "editCertificateCard": "certificates",
        "createCertificateCard": "certificates",
        "executeOperation": "operations",
        "alarmOperationWizard": "alarmOperationWizard",
        "createWorkgroup": "workgroup",
        "createDatamodelCard": "datamodel",
        "editDatamodelCard": "datamodel",
        "createArea": "areas",
        "updateArea": "areas",
        "wizardViews": "wizardViews",
        "manageOperationType": "manageOperationType",
        "createDataset": "datasets",
        "createTimeseries": "timeseries",
        "manageManufacturers": "manufacturers",
        "manageManufacturerModels": "models",
        "manageOrgManufacturers": "orgManufacturers",
        "manageOrgManufacturerModels": "orgModels",
        "createScheduler": "scheduler",
        "manageOrgSoftware": "orgSoftware",
        "createRestRequest": "restRequest",
        "createImageExecution": "imageExecution",
        "managePipelines": "pipelines",
        "manageOrganizationPlans": "organizationPlans",
        "manageDevicePlans": "devicePlans"
    },
    "priority": 1,
    "description": "description"
}
```

## Workspace metadata

| Field / Key | Value / Elements | Type | Description |
| :--- | :--- | :--- | :--- |
| **_id** | `_1696931515204_342` | String | Unique identifier |
| **name** | `Workspace title` | String | Workspace title |
| **description** | `description` | String | Workspace description |
| **icon** | `ogicon-abacus` | String | Assigned icon (font awesome 4 classes) |
| **color** | `#FF0000FF` | String (Hex RGBA) | Deprecated/Unused |
| **owner** | `josemaria.perez@amplia.es` | String (Email) | Owner email |
| **priority** | `1` | Integer | Priority level (stablish the dashboard show order 1 being the highest priority, 0 means don't show in the dashboards) |
| **users** | `[]` | Array  | List of users with readable access to the workspace. Always empty |
| **domains** | `[]` | Array | List of organizations with readable access to the workspace. Always empty |
| **workgroups** | `[]` | Array | Unused |
| **dashboards** | `[]` | Array | List of linked dashboards |
| **others.bannerImage** | `` | String (URL) | Banner URL Image |
| **others.showInHome** | `true` | Booleano | Show banner in home page |
| **others.mode** | `grid` or `list` or `carousel` | String | Dashboards layout mode in workspace page (grid, list, carousel) |
| **actions** | `[]` | Array (Strings) | Enabled actions while workspace is opened. Complete list in mapping of 'actions' below by default |
| **allowedProfiles** | `root`<br>`super_admin_domain`<br>`admin_domain`<br>`admin`<br>`advanced`<br>`viewer` | Array (Strings) | Profiles and roles authorized in workspace.|
| **widget_action** | `{}` | Object | Mapping of actions IDs with their modules. Complete list in mapping of 'widget_actions' below by default |

---

### Mapping of `actions`

| Action ID | Module |
| :--- | :--- |
| `editConnectorFunction` | `connectorFunctions` |
| `manageManufacturers` | `manufacturers` |
| `manageManufacturerModels` | `models` |
| `createArea` | `areas` |
| `createAsset` | `assets` |
| `createBundle` | `bundles` |
| `createCertificateCard` | `certificates` |
| `createChannel` | `channels` |
| `createDataset` | `datasets` |
| `createDevice` | `entities` |
| `executeOperation` | `operations` |
| `createDomain` | `domain` |
| `managePipelines` | `pipelines` |
| `createProvisionFunctions` | `provisionFunctions` |
| `createSubscriber` | `subscribers` |
| `createSubscription` | `subscriptions` |
| `createTicket` | `tickets` |
| `createTimeseries` | `timeseries` |
| `createUser` | `user` |
| `createWorkgroup` | `workgroup` |
| `createDatamodelCard` | `datamodel` |
| `deviceCollection` | `deviceCollection` |
| `manageOperationType` | `manageOperationType` |
| `wizardViews` | `wizardViews` |
| `editRuleConfiguration` | `ruleConfigurations` |
| `launchBulk` | `bulk` |
| `launchBulkProcessors` | `bulkProcessors` |
| `manageTransformers` | `transformers` |
| `manageAIModels` | `aiModels` |
| `manageDevicePlans` | `devicePlans` |
| `manageOrgSoftware` | `orgSoftware` |
| `createRestRequest` | `restRequest` |
| `manageOrganizationPlans` | `organizationPlans` |
| `manageOrgManufacturerModels` | `orgManufacturerModels` |
| `createImageExecution` | `imageExecution` |
| `manageOrgManufacturers` | `orgManufacturers` |
| `createTimeseriesFunctions` | `timeseriesFunctions` |
| `createScheduler` | `scheduler` |

### Mapping of `widget_action`

| Action Key | Module |
| :--- | :--- |
| `deviceCollection` | `deviceCollection` |
| `editConnectorFunction` | `connectorFunctions` |
| `editConnectorFunctionsCatalog` | `connectorFunctionsCatalog` |
| `editRuleConfiguration` | `ruleConfigurations` |
| `editDevice` | `entities` |
| `editAsset` | `assets` |
| `createDevice` | `entities` |
| `createAsset` | `assets` |
| `createTicket` | `tickets` |
| `editTicket` | `tickets` |
| `createSubscription` | `subscriptions` |
| `editSubscription` | `subscriptions` |
| `createSubscriber` | `subscribers` |
| `editSubscriber` | `subscribers` |
| `launchBulk` | `bulk` |
| `launchBulkProcessors` | `bulkProcessors` |
| `createProvisionFunctions` | `provisionFunctions` |
| `createTimeseriesFunctions` | `timeseriesFunction` |
| `createChannel` | `channels` |
| `createUser` | `user` |
| `editUserWizard` | `user` |
| `createDomain` | `domain` |
| `editBundle` | `bundles` |
| `createBundle` | `bundles` |
| `editCertificateCard` | `certificates` |
| `createCertificateCard` | `certificates` |
| `executeOperation` | `operations` |
| `alarmOperationWizard` | `alarmOperationWizard` |
| `createWorkgroup` | `workgroup` |
| `createDatamodelCard` | `datamodel` |
| `editDatamodelCard` | `datamodel` |
| `createArea` | `areas` |
| `updateArea` | `areas` |
| `wizardViews` | `wizardViews` |
| `manageOperationType` | `manageOperationType` |
| `createDataset` | `datasets` |
| `createTimeseries` | `timeseries` |
| `manageManufacturers` | `manufacturers` |
| `manageManufacturerModels` | `models` |
| `manageOrgManufacturers` | `orgManufacturers` |
| `manageOrgManufacturerModels` | `orgModels` |
| `createScheduler` | `scheduler` |
| `manageOrgSoftware` | `orgSoftware` |
| `createRestRequest` | `restRequest` |
| `createImageExecution` | `imageExecution` |
| `managePipelines` | `pipelines` |
| `manageOrganizationPlans` | `organizationPlans` |
| `manageDevicePlans` | `devicePlans` |

