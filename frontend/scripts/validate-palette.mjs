// Palette checks: OKLab distance between every pair, the same under the
// three dichromacies, and WCAG contrast against the surface. Colour is
// the computable part of design, so it gets computed rather than judged.

const hex = h => { const n = parseInt(h.replace('#',''),16); return [(n>>16&255)/255,(n>>8&255)/255,(n&255)/255]; };
const lin = c => c <= 0.04045 ? c/12.92 : Math.pow((c+0.055)/1.055, 2.4);
const srgbLin = c => lin(c);

function oklab([r,g,b]) {
  const R=srgbLin(r), G=srgbLin(g), B=srgbLin(b);
  const l = Math.cbrt(0.4122214708*R + 0.5363325363*G + 0.0514459929*B);
  const m = Math.cbrt(0.2119034982*R + 0.6806995451*G + 0.1073969566*B);
  const s = Math.cbrt(0.0883024619*R + 0.2817188376*G + 0.6299787005*B);
  return [0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
          1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
          0.0259040371*l + 0.7827717662*m - 0.8086757660*s];
}
const dE = (a,b) => { const A=oklab(a), B=oklab(b);
  return Math.hypot(A[0]-B[0], A[1]-B[1], A[2]-B[2]) * 100; };

// Machado et al. severity-1.0 CVD matrices.
const CVD = {
  deuter: [[0.367322,0.860646,-0.227968],[0.280085,0.672501,0.047413],[-0.011820,0.042940,0.968881]],
  protan: [[0.152286,1.052583,-0.204868],[0.114503,0.786281,0.099216],[-0.003882,-0.048116,1.051998]],
  tritan: [[1.255528,-0.076749,-0.178779],[-0.078411,0.930809,0.147602],[0.004733,0.691367,0.303900]],
};
const sim = (rgb, m) => m.map(row => Math.min(1, Math.max(0, row[0]*rgb[0] + row[1]*rgb[1] + row[2]*rgb[2])));

const relLum = ([r,g,b]) => 0.2126*srgbLin(r) + 0.7152*srgbLin(g) + 0.0722*srgbLin(b);
const contrast = (a,b) => { const L1=relLum(a), L2=relLum(b);
  return (Math.max(L1,L2)+0.05)/(Math.min(L1,L2)+0.05); };

export function report(name, colours, surface) {
  const names = Object.keys(colours);
  const rgbs = names.map(n => hex(colours[n]));
  const surf = hex(surface);
  console.log(`\n=== ${name} (surface ${surface}) ===`);

  let worstNormal = Infinity, worstCvd = Infinity, worstPair = '', worstCvdPair = '';
  for (let i = 0; i < names.length; i++) {
    for (let j = i+1; j < names.length; j++) {
      const n = dE(rgbs[i], rgbs[j]);
      if (n < worstNormal) { worstNormal = n; worstPair = `${names[i]}/${names[j]}`; }
      for (const [k, m] of Object.entries(CVD)) {
        const c = dE(sim(rgbs[i], m), sim(rgbs[j], m));
        if (c < worstCvd) { worstCvd = c; worstCvdPair = `${names[i]}/${names[j]} (${k})`; }
      }
    }
  }
  const flag = (v, floor) => v >= floor ? 'PASS' : 'FAIL';
  console.log(`  closest pair, normal vision : ${worstNormal.toFixed(1)}  ${worstPair}  [${flag(worstNormal,15)} floor 15]`);
  console.log(`  closest pair, colour-blind  : ${worstCvd.toFixed(1)}  ${worstCvdPair}  [${flag(worstCvd,8)} target 8]`);
  for (const n of names) {
    const c = contrast(hex(colours[n]), surf);
    const mark = c >= 4.5 ? 'AA  ' : c >= 3 ? 'AA-lg' : 'low ';
    console.log(`  ${n.padEnd(14)} ${colours[n]}  contrast ${c.toFixed(2)}  ${mark}`);
  }
}

// The palette this repository actually ships, so the checks can be
// re-run after any change to tokens.css rather than trusted to memory.
//   node scripts/validate-palette.mjs
const LIGHT = "#ffffff", DARK = "#15131c";
report("ad-type ramp, light", { influencer:"#df0d66", display:"#804d06", performance:"#769306",
  video:"#079b98", "google-ads":"#1864f6", social:"#7415bf" }, LIGHT);
report("ad-type ramp, dark", { influencer:"#f34d95", display:"#fdbd0a", performance:"#8c9f02",
  video:"#1cdbb3", "google-ads":"#6e7cfd", social:"#d3b5ff" }, DARK);
report("status, light", { good:"#016807", warn:"#ae6408", crit:"#9c022b" }, LIGHT);
report("status, dark",  { good:"#87eb60", warn:"#ce8703", crit:"#fe868c" }, DARK);
// Text roles and the accent are checked for contrast against their
// background, not for separation from each other. The pairwise check
// above is for marks that sit side by side in a chart and have to be
// told apart; secondary text and a link colour are distinguished by
// position and weight, and holding them to a series floor would report
// a failure that means nothing.
report("text on surface, light", { ink:"#15121c", ink2:"#3a3548", muted:"#655f74" }, LIGHT);
report("text on surface, dark",  { ink:"#efecf5", ink2:"#bdb7c9", muted:"#8e879f" }, DARK);
report("accent, light", { accent:"#6025d9" }, LIGHT);
report("accent, dark",  { accent:"#b79cff" }, DARK);
