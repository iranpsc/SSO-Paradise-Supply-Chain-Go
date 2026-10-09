"""One-time, deterministic port of the owned Laravel shell to typed React markup.

Run from the Go repository root. Does not read .env or alter Laravel files.
"""
from pathlib import Path
from html.parser import HTMLParser
import json
import re
import shutil

root = Path(__file__).resolve().parents[1]
source = root.parent / "SSO-Paradise-Supply-Chain"
template = (source / "resources/views/components/layouts/app.blade.php").read_text(encoding="utf-8")
frontend = root.parent / "SSO-Paradise-Supply-Chain-Go-Front"
destination = frontend / "src/components/layout"
destination.mkdir(parents=True, exist_ok=True)

class JSX(HTMLParser):
    void = {"img", "input", "br", "hr", "meta", "link"}
    names = {"class": "className", "for": "htmlFor", "srcset": "srcSet", "tabindex": "tabIndex", "viewbox": "viewBox", "fill-rule": "fillRule", "clip-rule": "clipRule", "stroke-width": "strokeWidth", "stroke-linecap": "strokeLinecap", "stroke-linejoin": "strokeLinejoin", "xlink:href": "xlinkHref", "xmlns:xlink": "xmlnsXlink", "patterncontentunits": "patternContentUnits", "enable-background": "enableBackground"}
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.parts = []
    def handle_starttag(self, tag, attrs):
        if tag == "account-menu":
            self.parts.append("<AccountMenu />")
            return
        if tag == "session-control":
            self.parts.append("<SessionControl />")
            return
        props = []
        values = dict(attrs)
        for k, v in attrs:
            if k == "style" or (k == "for" and tag != "label"):
                continue
            if k == "onclick":
                props.append("onClick={" + ("openNav" if "openNav2" in v else "closeNav") + "}")
                props.extend(['role="button"', 'tabIndex={0}', 'aria-label="' + ('باز کردن فهرست' if 'openNav2' in v else 'بستن فهرست') + '"', 'onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); ' + ('openNav' if 'openNav2' in v else 'closeNav') + '(); } }}'])
                continue
            if k == "href" and v in {"./index.html", "#"} and values.get("aria-label") == "logo":
                v = "/home"
            if k == "class":
                v = v.replace("!z-[5000]", "z-[5000]")
                if "enable-dark-mode" in v:
                    props.append('onClick={() => setTheme(true)}')
                if "disable-dark-mode" in v:
                    props.append('onClick={() => setTheme(false)}')
            if v is None:
                props.append(self.names.get(k, k))
            else:
                props.append(self.names.get(k, k) + "={" + json.dumps(v, ensure_ascii=False) + "}")
        if tag == "a" and values.get("target") == "_blank":
            props.append('rel="noopener noreferrer"')
        self.parts.append("<" + tag + (" " if props else "") + " ".join(props) + (" />" if tag in self.void else ">"))
    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)
        if tag not in self.void:
            self.handle_endtag(tag)
    def handle_endtag(self, tag):
        if tag not in self.void and tag not in {"account-menu", "session-control"}:
            self.parts.append("</" + tag + ">")
    def handle_data(self, data):
        if data.strip():
            self.parts.append("{" + json.dumps(data, ensure_ascii=False) + "}")
        else:
            self.parts.append("\n")

def prepare(text):
    text = re.sub(r"\{\{--.*?--\}\}", "", text, flags=re.S)
    text = re.sub(r"\{\{\s*asset\('([^']+)'\)\s*\}\}", r"/\1", text)
    return text

header = template[template.index("<header>"):template.index("</header>") + 9]
header = prepare(header)
# The first authenticated area is the account navigation, the other two are logout controls.
auth_index = [0]
def auth_replace(match):
    auth_index[0] += 1
    return "<account-menu></account-menu>" if auth_index[0] == 1 else ""
header = re.sub(r"@auth\b.*?@endauth", auth_replace, header, flags=re.S)
header = re.sub(r"@guest\b.*?@endguest", "<session-control></session-control>", header, flags=re.S)
parser = JSX(); parser.feed(header)
(destination / "legacy-header.tsx").write_text('"use client";\nimport { AccountMenu, SessionControl } from "./session-controls";\n// Markup and artwork preserved from the Laravel layout; interactions are React-owned.\nexport function LegacyHeader({ openNav, closeNav, setTheme }: { openNav: () => void; closeNav: () => void; setTheme: (dark: boolean) => void }) {\nreturn (' + "".join(parser.parts) + ');\n}\n', encoding="utf-8")
footer = prepare(template[template.index("<footer"):template.index("</footer>") + 9])
parser = JSX(); parser.feed(footer)
(destination / "legacy-footer.tsx").write_text('// Content, links, classes and artwork preserved from the Laravel footer.\nexport function LegacyFooter() { return (' + "".join(parser.parts) + '); }\n', encoding="utf-8")
for folder in ["images/logo", "style/fonts"]:
    shutil.copytree(source / "public" / folder, frontend / "public" / folder, dirs_exist_ok=True)
assert "{{" not in (destination / "legacy-header.tsx").read_text(encoding="utf-8")
print("Ported original header/footer and local brand assets.")
