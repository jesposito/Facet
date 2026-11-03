# Unraid

Facet is available in Unraid Community Applications. Open **Apps**, search for **Facet**, and install it.

To add it by hand instead, go to **Docker → Add Container** and use this template URL:

```
https://raw.githubusercontent.com/jesposito/unraid_templates/main/templates/facet.xml
```

The template is maintained in [jesposito/unraid_templates](https://github.com/jesposito/unraid_templates), which is its only source. This repository used to carry a copy in `unraid/facet-template.xml`, but it drifted out of date and was removed.

For the settings to choose (data and uploads paths, `ADMIN_EMAILS`, `APP_URL`, `TRUST_PROXY`, `PUID`/`PGID`) and how to back up, see [Install → Unraid](../docs/SETUP.md#option-4-unraid) and [Upgrades, backups and recovery](../docs/UPGRADE.md).
