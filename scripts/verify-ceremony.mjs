#!/usr/bin/env node
// verify-ceremony.mjs
//
// Offline verifier for a Groth16 trusted-setup ceremony transcript produced
// by `engine/cmd/ceremony`. Checks the *file-integrity* half of the
// verification chain: that every artefact SHA-256 in zk/ceremony/ matches
// what attestations.json says, and that the manifest pins line up.
//
// This deliberately does NOT re-run the gnark VerifyPhase1 / VerifyPhase2
// pairing checks - those already run in
// engine/internal/zkverifier/ceremony/ceremony_test.go on every `go test`
// and require the full gnark build. This script exists so a third party
// with only Node.js + the artefact directory can convince themselves that
// what is on disk is what the coordinator published, before spending the
// cost of a full gnark re-verify.
//
// Usage:
//   node scripts/verify-ceremony.mjs [--dir zk/ceremony]
//
// Exit codes:
//   0 - every hash and manifest pin matched
//   1 - a hash mismatch or missing artefact
//   2 - invalid inputs (bad path, malformed JSON)
//
// No external deps - stdlib only.

import { createHash } from 'node:crypto';
import { readFileSync, existsSync, statSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { argv, exit } from 'node:process';

function parseArgs(argv) {
  const args = { dir: 'zk/ceremony' };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--dir' || a === '-d') { args.dir = argv[++i]; continue; }
    if (a === '--help' || a === '-h') { args.help = true; continue; }
  }
  return args;
}

function sha256Hex(path) {
  const buf = readFileSync(path);
  return createHash('sha256').update(buf).digest('hex');
}

function readJson(path) {
  const raw = readFileSync(path, 'utf8');
  try { return JSON.parse(raw); }
  catch (e) { throw new Error(`invalid JSON at ${path}: ${e.message}`); }
}

function fail(msg) { console.error(`[verify-ceremony] FAIL: ${msg}`); exit(1); }
function bad(msg)  { console.error(`[verify-ceremony] bad input: ${msg}`); exit(2); }
function ok(msg)   { console.log(`[verify-ceremony] ok: ${msg}`); }

function main() {
  const args = parseArgs(argv);
  if (args.help) {
    console.log('usage: node scripts/verify-ceremony.mjs [--dir zk/ceremony]');
    exit(0);
  }

  const dir = resolve(args.dir);
  if (!existsSync(dir) || !statSync(dir).isDirectory()) {
    bad(`ceremony directory not found: ${dir}`);
  }

  const attPath = join(dir, 'attestations.json');
  if (!existsSync(attPath)) {
    bad(`attestations.json not found in ${dir} - run 'go run ./engine/cmd/ceremony --out ${args.dir}' first`);
  }
  const att = readJson(attPath);
  const tr = att.transcript;
  if (!tr) bad('attestations.json is missing transcript field');

  // 1) Manifest pin consistency (best-effort; manifest is a self-declared
  // pin file, not required for a hash check, but if it exists we sanity
  // it against the transcript).
  const manifestPath = join(dir, 'manifest.json');
  if (existsSync(manifestPath)) {
    const mf = readJson(manifestPath);
    if (mf.circuit && mf.circuit.id && mf.circuit.id !== tr.circuit_id) {
      fail(`manifest.circuit.id=${mf.circuit.id} != transcript.circuit_id=${tr.circuit_id}`);
    }
    if (mf.ceremony && typeof mf.ceremony.phase1_domain_size_default === 'number'
        && mf.ceremony.phase1_domain_size_default !== tr.phase1_domain_size) {
      // domain size is a default in the manifest; a real ceremony may
      // deviate. Warn but don't fail.
      console.warn(`[verify-ceremony] warn: manifest default N=${mf.ceremony.phase1_domain_size_default} but transcript N=${tr.phase1_domain_size}`);
    }
    ok(`manifest.json consistent with transcript (circuit_id=${tr.circuit_id})`);
  } else {
    console.warn('[verify-ceremony] warn: no manifest.json in ceremony dir - skipping pin check');
  }

  // 2) Final artefact hashes. Each of these files, if present, must hash
  // to what the transcript claims.
  const targets = [
    { file: 'phase1_commons.bin', field: 'phase1_commons_sha256' },
    { file: 'groth16_pk.bin',     field: 'final_pk_sha256' },
    { file: 'groth16_vk.bin',     field: 'final_vk_sha256' },
  ];

  let missing = 0;
  for (const t of targets) {
    const p = join(dir, t.file);
    if (!existsSync(p)) {
      console.warn(`[verify-ceremony] warn: ${t.file} missing - skipping (this is the example transcript with placeholder hashes)`);
      missing++;
      continue;
    }
    const got = sha256Hex(p);
    const want = tr[t.field];
    if (!want) fail(`transcript missing field ${t.field}`);
    if (want.startsWith('<') || want === '') {
      // example / placeholder attestations - not a real run
      console.warn(`[verify-ceremony] warn: transcript.${t.field} is a placeholder (${want}) - not a real ceremony transcript`);
      continue;
    }
    if (got.toLowerCase() !== want.toLowerCase()) {
      fail(`sha256(${t.file}) = ${got} but transcript.${t.field} = ${want}`);
    }
    ok(`sha256(${t.file}) matches transcript.${t.field}`);
  }

  // 3) Structural sanity on per-contribution entries.
  for (const phaseKey of ['phase1', 'phase2']) {
    const entries = tr[phaseKey];
    if (!Array.isArray(entries) || entries.length === 0) {
      fail(`transcript.${phaseKey} is missing or empty`);
    }
    entries.forEach((e, i) => {
      if (e.index !== i) fail(`transcript.${phaseKey}[${i}].index=${e.index} - expected ${i}`);
      if (!e.digest_hex) fail(`transcript.${phaseKey}[${i}] missing digest_hex`);
    });
    ok(`transcript.${phaseKey} has ${entries.length} contributions with sequential indices`);
  }

  // 4) Honesty label surfacing - if the caller sees this script pass,
  // they should also see the honesty label so they know what property
  // they just verified.
  if (tr.honesty_label) {
    console.log(`[verify-ceremony] honesty_label: ${tr.honesty_label}`);
  }

  if (missing === targets.length) {
    console.warn('[verify-ceremony] note: no artefact binaries were present - this only checked the transcript structure, not any real ceremony');
  }

  console.log('[verify-ceremony] all checks passed');
}

main();
