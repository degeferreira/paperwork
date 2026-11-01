// Deterministic, dependency-free ZIP (stored entries, fixed DOS epoch).
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const root = path.join(__dirname, '../zotero');
const {version} = JSON.parse(fs.readFileSync(path.join(root, 'manifest.json'), 'utf8'));
function crc32(bytes) {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}
const local = [], central = [];
let offset = 0;
for (const filename of ['manifest.json', 'bootstrap.js', 'aliases.js']) {
  const name = Buffer.from(filename), data = fs.readFileSync(path.join(root, filename));
  const header = Buffer.alloc(30);
  header.writeUInt32LE(0x04034b50); header.writeUInt16LE(20, 4);
  header.writeUInt16LE(33, 12); header.writeUInt32LE(crc32(data), 14);
  header.writeUInt32LE(data.length, 18); header.writeUInt32LE(data.length, 22); header.writeUInt16LE(name.length, 26);
  local.push(header, name, data);
  const entry = Buffer.alloc(46);
  entry.writeUInt32LE(0x02014b50); entry.writeUInt16LE(20, 4); entry.writeUInt16LE(20, 6);
  header.copy(entry, 8, 6, 28); entry.writeUInt32LE(offset, 42);
  central.push(entry, name); offset += header.length + name.length + data.length;
}
const directory = Buffer.concat(central), end = Buffer.alloc(22);
end.writeUInt32LE(0x06054b50); end.writeUInt16LE(3, 8); end.writeUInt16LE(3, 10);
end.writeUInt32LE(directory.length, 12); end.writeUInt32LE(offset, 16);
const archive = Buffer.concat([...local, directory, end]);
const filename = `paperwork-zotero-${version}.xpi`;
fs.mkdirSync(path.join(root, 'dist'), {recursive: true});
fs.writeFileSync(path.join(root, 'dist', filename), archive);
fs.writeFileSync(path.join(root, 'dist', filename + '.sha256'), crypto.createHash('sha256').update(archive).digest('hex') + '  ' + filename + '\n');
console.log(filename);
