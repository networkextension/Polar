# Create/update one "<project> — docs" Plane wiki page per project from a JSON map.
# Pages are NOT in Plane's public API, so we go through the Django ORM. Run on the
# Plane host from apps/api:
#   PLANE_PAGES_JSON=pages.json PLANE_WS_SLUG=<slug> \
#     python manage.py shell -c "exec(open('scripts/plane-orm-page.py').read())"
#
# Input: env PLANE_PAGES_JSON → {"<project-name-with-hyphens>": "<description_html>"}.
#   Plane project names use spaces (Plane forbids '-' in names); we match each entry
#   by project.name.replace(" ", "-").
# Setting description_html (binary=None) is enough — the `live`/hocuspocus service
# rebuilds the collaborative binary from the HTML on first open. This is the same
# shape Plane's own page-create endpoint produces.
import os, json
from django.utils.html import strip_tags
from plane.db.models import Workspace, Project, Page, ProjectPage

SLUG = os.environ.get("PLANE_WS_SLUG", "default")
PAGE_SUFFIX = os.environ.get("PLANE_PAGE_SUFFIX", " — docs")
docs = json.load(open(os.environ["PLANE_PAGES_JSON"]))
ws = Workspace.objects.get(slug=SLUG)
owner = ws.owner

for p in Project.objects.filter(workspace=ws).order_by("name"):
    html = docs.get(p.name.replace(" ", "-"))
    if not html:
        continue
    name = f"{p.name}{PAGE_SUFFIX}"
    stripped = strip_tags(html)[:50000]
    pp = ProjectPage.objects.filter(project=p, page__name=name).select_related("page").first()
    if pp:
        pg = pp.page
        pg.description_html = html; pg.description_binary = None
        pg.description_json = {}; pg.description_stripped = stripped; pg.save()
        print("updated", p.name, f"{len(html)//1024}KB")
    else:
        pg = Page.objects.create(workspace=ws, name=name, description_html=html,
                                 description_stripped=stripped, owned_by=owner,
                                 created_by=owner, updated_by=owner, access=0)
        ProjectPage.objects.create(workspace=ws, project=p, page=pg,
                                   created_by=owner, updated_by=owner)
        print("created", p.name, f"{len(html)//1024}KB")
