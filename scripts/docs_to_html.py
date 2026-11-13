"""Turn docs/*.md into the SDK's HTML pages.

    python scripts/docs_to_html.py <docs folder> <SDK folder>

Writes <SDK>/docs/<NAME>.html for every page, <SDK>/docs/index.html listing
them, and <SDK>/Start Here.html from GETTING_STARTED.md. Links between pages
point at the .html files. Needs the `markdown` package (pip install markdown).
"""
import html
import pathlib
import re
import shutil
import sys

try:
    import markdown
except ImportError:
    sys.exit("docs_to_html: the markdown package is missing; run: pip install markdown")

CSS = """
:root { --bg:#ffffff; --fg:#1d2330; --muted:#5d6678; --line:#e3e6ec; --code:#f4f5f8; --accent:#3b6fd8; }
@media (prefers-color-scheme: dark) {
  :root { --bg:#16191f; --fg:#e4e7ee; --muted:#9aa3b5; --line:#2b3039; --code:#1f232b; --accent:#7ea6ff; }
}
* { box-sizing: border-box; }
body { margin:0; background:var(--bg); color:var(--fg);
  font: 16px/1.6 "Segoe UI", system-ui, -apple-system, sans-serif; }
header { border-bottom:1px solid var(--line); padding:12px 24px; display:flex; gap:20px; align-items:center; flex-wrap:wrap; }
header .brand { font-weight:700; letter-spacing:.2px; color:var(--fg); text-decoration:none; }
header a { color:var(--muted); text-decoration:none; font-size:14px; }
header a:hover { color:var(--accent); }
main { max-width:860px; margin:0 auto; padding:24px 24px 64px; }
h1 { font-size:2rem; line-height:1.2; margin:.6em 0 .5em; }
h2 { font-size:1.4rem; margin:2em 0 .5em; padding-top:.4em; border-top:1px solid var(--line); }
h3 { font-size:1.1rem; margin:1.6em 0 .4em; }
a { color:var(--accent); }
code { font-family: "Cascadia Mono", Consolas, monospace; font-size:.9em; background:var(--code); padding:.1em .35em; border-radius:4px; }
pre { background:var(--code); padding:14px 16px; border-radius:8px; overflow-x:auto; line-height:1.45; }
pre code { background:none; padding:0; font-size:.88em; }
table { border-collapse:collapse; width:100%; margin:1em 0; display:block; overflow-x:auto; }
th, td { border:1px solid var(--line); padding:6px 10px; text-align:left; vertical-align:top; }
th { background:var(--code); }
img { max-width:100%; border-radius:6px; }
blockquote { margin:1em 0; padding:.2em 1em; border-left:3px solid var(--accent); color:var(--muted); }
ul.pages { columns:2; padding-left:1.2em; }
@media (max-width:640px) { ul.pages { columns:1; } main { padding:16px; } }
"""

PAGE = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title} · moonBASIC</title>
<style>{css}</style>
</head>
<body>
<header>
  <a class="brand" href="{home}">moonBASIC</a>
  <a href="{home}">Start here</a>
  <a href="{docs}index.html">All docs</a>
  <a href="{docs}GUIDE.html">Guide</a>
  <a href="{docs}COMMANDS.html">Commands</a>
</header>
<main>
{body}
</main>
</body>
</html>
"""

LINK = re.compile(r'href="([^":#?]+)\.md(#[^"]*)?"')


def render(md_text, link_prefix):
    conv = markdown.Markdown(extensions=["tables", "fenced_code", "toc", "sane_lists"])
    body = conv.convert(md_text)

    def relink(m):
        target, anchor = m.group(1), m.group(2) or ""
        if "/" in target:  # ../README.md and the like are not shipped
            return 'href="%s.md%s"' % (target, anchor)
        return 'href="%s%s.html%s"' % (link_prefix, target, anchor)

    body = LINK.sub(relink, body)
    if link_prefix:
        # On the root page, ../examples/x.mb is examples/x.mb
        body = body.replace('href="../', 'href="')
    return body


def title_of(md_text, fallback):
    m = re.search(r"^#\s+(.+)$", md_text, re.M)
    return m.group(1).strip() if m else fallback


def main():
    src = pathlib.Path(sys.argv[1])
    out = pathlib.Path(sys.argv[2])
    docs_out = out / "docs"
    docs_out.mkdir(parents=True, exist_ok=True)
    pages = []
    for md in sorted(src.glob("*.md")):
        text = md.read_text(encoding="utf-8")
        title = title_of(text, md.stem)
        body = render(text, "")
        (docs_out / (md.stem + ".html")).write_text(
            PAGE.format(title=html.escape(title), css=CSS, home="../Start%20Here.html", docs="", body=body),
            encoding="utf-8")
        pages.append((md.stem, title))
    if (src / "images").is_dir():
        shutil.copytree(src / "images", docs_out / "images", dirs_exist_ok=True)

    first = ["GETTING_STARTED", "GUIDE", "LANGUAGE", "COMMANDS", "STUDIO", "RELEASE"]
    ordered = [p for f in first for p in pages if p[0] == f] + [p for p in pages if p[0] not in first]
    items = "\n".join('<li><a href="%s.html">%s</a></li>' % (n, html.escape(t)) for n, t in ordered)
    body = "<h1>moonBASIC documentation</h1>\n<p>Start with the guide, then the topic you need.</p>\n<ul class=\"pages\">\n%s\n</ul>" % items
    (docs_out / "index.html").write_text(
        PAGE.format(title="Documentation", css=CSS, home="../Start%20Here.html", docs="", body=body),
        encoding="utf-8")

    start = (src / "GETTING_STARTED.md").read_text(encoding="utf-8")
    body = render(start, "docs/")
    (out / "Start Here.html").write_text(
        PAGE.format(title="Start here", css=CSS, home="Start%20Here.html", docs="docs/", body=body),
        encoding="utf-8")
    print("docs_to_html: %d pages, index and Start Here.html" % len(pages))


if __name__ == "__main__":
    main()
