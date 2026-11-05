# Navigation and role authorization

Applications register trusted `NavigationDefinition` data with `services.RegisterNavigation`, then call `NavigationServiceFor(db).Initialize()`. Register business permissions with `PermissionService.SyncCatalog` before `InitializeCompanyRoles`; `RefreshRoleGrants` updates API expansion after route registration. Register the authenticated navigation routes with `routers.InitNavigationRouter(publicGroup)` alongside the existing role router. Registration must finish before accepting requests.

The foundation owns `sys_menu` and normalized `sys_role_menu` relations. Menu records store presentation and reference registered targets; callers cannot supply routes, access requirements or action definitions. Only the fixed platform `admin` identity manages global navigation. Protected targets and their containing groups remain visible.

Role details use `POST /role`. Roles are scoped to one company, codes are unique within that company, and administrator identity is never assignable through role codes. Role details and role authorization are independent mutations.

`GET /role/:guid/authorization` returns the role, company name, complete menu catalog, registered business actions, selected menus and effective permissions. Each menu includes `assignable`, `defaultVisible` and `restrictionReason`; clients use these fields for interaction restrictions.

`PUT /role/:guid/authorization` accepts only `menuKeys` and `permissionGuids`. The service checks the actor, company scope, registered targets, required identities and exact allowed actions. It adds menu read permissions, expands dependencies and APIs, and replaces both permission and menu relations in one transaction. Ordinary roles cannot receive platform menus, identity-only administration menus or arbitrary API permissions. Saving authorization never changes role details.

`GET /navigation` returns visible entries selected by enabled company roles, then checks their required identity and effective permissions. Multiple roles contribute a union of menu selections. Query dependencies grant API access without adding menu entries. Authenticated public entries are always available. The platform administrator bypasses company grants while respecting presentation visibility and target-specific display rules.

Built-in company roles receive initial menus once their default permissions exist. Explicit empty selections remain empty across restarts. Newly created custom roles start with empty permissions and menus. Deleting a custom role requires removing its users and also clears both kinds of grants; deleting a custom menu clears its menu relations.

The old application-specific navigation tables and separate role permission/editor endpoints are not read or migrated.
