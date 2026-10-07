'use strict';
const views = {
 home: ['Your adventure starts here.', 'A Town Map, your trainer card and the next place to explore. Everything is one selection away.', 'PokéCRT Home screen showing the Town Map and Adventure Menu'],
 pokedex: ['A collection, one discovery at a time.', 'Browse collected appearances, evolution families and encounter records in your private Pokédex.', 'Pokédex showing Scyther artwork, the National Index and discovery details'],
 encounter: ['Meet your next discovery.', 'Record an encounter, discover an appearance and earn progression. Review recent discoveries without creating a new encounter.', 'Encounter result with Pokémon artwork, XP and progression details'],
 trainer: ['Your own local adventure.', 'Keep track of your trainer level, statistics and generation collection progress. Multiple local trainers can have their own journeys.', 'Trainer card and collection progress by generation'],
 achievements: ['Always something to work toward.', 'See your earned badges and your next collection goals across 50 achievements.', 'Achievements page showing earned badges and next collection goals']
};
const announce = message => {document.getElementById('announcement').textContent = message;};
async function copyText(text, button) {
 const original = button.textContent;
 try {
  if (!navigator.clipboard || !window.isSecureContext) throw new Error('Clipboard unavailable');
  await navigator.clipboard.writeText(text);button.textContent = 'Copied';announce('Command copied to clipboard.');
 } catch {
  button.textContent = 'Select';announce('Clipboard unavailable. Select and copy the displayed command.');
  const target = button.id === 'copy-install' ? document.getElementById('install-text') : button.parentElement.querySelector('code');
  const range = document.createRange();range.selectNodeContents(target);const selection = window.getSelection();selection.removeAllRanges();selection.addRange(range);
 }
 setTimeout(() => {button.textContent = original;}, 2000);
}
function wireTabs(container, onSelect) {
 const tabs = [...container.querySelectorAll('[role="tab"]')];
 const select = tab => {tabs.forEach(t => {t.setAttribute('aria-selected', String(t === tab));t.tabIndex = t === tab ? 0 : -1;});onSelect(tab);};
 tabs.forEach(tab => {
  tab.addEventListener('click', () => select(tab));
  tab.addEventListener('keydown', event => {
   let index = tabs.indexOf(tab);
   if(event.key === 'ArrowRight') index = (index + 1) % tabs.length;
   else if(event.key === 'ArrowLeft') index = (index - 1 + tabs.length) % tabs.length;
   else if(event.key === 'Home') index = 0;
   else if(event.key === 'End') index = tabs.length - 1;
   else return;
   event.preventDefault();select(tabs[index]);tabs[index].focus();
  });
 });
}
wireTabs(document.querySelector('.showcase-tabs'), tab => {
 const key = tab.dataset.screen;const [title, text, alt] = views[key];
 const img = document.getElementById('screen-image');img.src = `assets/${key}.png`;img.alt = alt;
 document.getElementById('screen-title').textContent = title;document.getElementById('screen-copy').textContent = text;
 document.getElementById('screen-panel').setAttribute('aria-labelledby',tab.id);
});
const source = document.getElementById('install-text').textContent;
const archive = `# Download the archive and SHA256SUMS from one release.
# In the directory containing those downloaded files:
sha256sum -c SHA256SUMS
# Keep one matching archive in this directory.
tar -xzf pokecrt_*_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 0755 pokecrt "$HOME/.local/bin/pokecrt"
"$HOME/.local/bin/pokecrt" --version
"$HOME/.local/bin/pokecrt" tui`;
wireTabs(document.querySelector('.install-tabs'), tab => {
 const isSource = tab.dataset.install === 'source';
 document.getElementById('install-text').textContent = isSource ? source : archive;
 document.getElementById('install-label').textContent = isSource ? 'Build the current source / Go 1.27+' : 'Install a downloaded release / Linux amd64';
 document.getElementById('install-note').textContent = isSource ? 'Internet is needed to clone and prepare assets. Go 1.27+ is a build requirement, not a requirement for the packaged executable.' : 'Keep only one matching Linux amd64 archive in the download directory. If ~/.local/bin is not on PATH, add it to your shell configuration. Keep the extracted documentation for reference.';
 document.getElementById('install-panel').setAttribute('aria-labelledby',tab.id);
});
document.querySelectorAll('[data-copy]').forEach(button => button.addEventListener('click', () => copyText(button.dataset.copy,button)));
document.getElementById('copy-install').addEventListener('click', event => copyText(document.getElementById('install-text').textContent,event.currentTarget));
