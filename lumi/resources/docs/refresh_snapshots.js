const fs = require("fs");
const path = require("path");

const snapFolder = "./static";

const files = fs.readdirSync(snapFolder);
const snaps = files.filter(f => f.match(/\+\d+.json/))

const res = {}
snaps.forEach((f) => {
  let raw = fs.readFileSync(path.join(snapFolder, f), {encoding: "utf-8"})
  let base = f.replace(".json", "")
  res[base] = JSON.parse(raw)
})

console.log('found '+Object.keys(res).length+' resources')

const dst = path.join(snapFolder, "snapshots.json")
fs.writeFileSync(dst, JSON.stringify(res), {encoding: "utf-8"})
console.log('saved to '+dst)
