// Generated JavaScript: edit this TypeScript source and run npm run build.
type AliasRegistry = Record<string, string>;
interface AliasCreator { lastName?: string; name?: string }
interface AliasMetadata { creators?: AliasCreator[]; date?: string; title?: string }
/* Pure alias rules shared by the Zotero bootstrap and contract tests. */
var PaperworkAliases = (() => {
  const pattern = /^[A-Za-z0-9][A-Za-z0-9-]{0,99}$/;
  function ascii(text: unknown) {return String(text || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/[^A-Za-z0-9]/g, '');}
  function read(extra: string) {
    const lines = String(extra || '').split(/\r?\n/).filter(line => /^Paperwork Alias:/i.test(line));
    if (lines.length > 1) throw new Error('Mais de um Paperwork Alias no mesmo item. Corrija antes de continuar.');
    if (!lines.length) return null;
    const alias = lines[0].split(':').slice(1).join(':').trim();
    if (!pattern.test(alias)) throw new Error('Paperwork Alias inválido no campo Extra.');
    return alias;
  }
  function base({creators, date, title}: AliasMetadata) {
    const author = ascii(creators?.[0]?.lastName || creators?.[0]?.name || 'Anon').slice(0,12) || 'Anon';
    const year = String(date || '').match(/(?:18|19|20|21)\d{2}/)?.[0]?.slice(-2) || 'SD';
    const stop = new Set(['a','o','as','os','um','uma','de','da','do','das','dos','e','em','the','of','an','and','on','for']);
    const word = String(title || '').split(/\s+/).find(w => ascii(w) && !stop.has(ascii(w).toLowerCase())) || 'Obra';
    const short = ascii(word).slice(0,3); return author+year+'-'+(short || 'Obra');
  }
  function reserve(registry: AliasRegistry, alias: string, identity: string) {
    if (!pattern.test(alias)) throw new Error('Alias inválido.');
    if (Object.hasOwn(registry,alias) && registry[alias]!==identity) throw new Error('Alias já reservado para outra identidade: '+alias);
    registry[alias] = identity; return alias;
  }
  function allocate(registry: AliasRegistry, stem: string, identity: string) {
    let alias=stem, n=2;
    while (Object.hasOwn(registry,alias) && registry[alias]!==identity) alias=stem+'-'+n++;
    return reserve(registry,alias,identity);
  }
  return {read, base, reserve, allocate};
})();
