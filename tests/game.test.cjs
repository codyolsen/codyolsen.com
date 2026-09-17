const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { test } = require('node:test');
const vm = require('node:vm');

// Run the real game with deterministic browser/audio substitutes. The test-only
// accessor exposes resource counts without shipping a debug API to the browser.
function game(mobile = true, audioAvailable = true) {
  let now = 0;
  let nextId = 0;
  const frames = new Map();
  const nodes = [];
  const buffers = [];
  const media = [];
  const gradients = [];
  function element() {
    const events = new Map();
    return {
      style: {}, classList: { add() {} }, captures: new Set(),
      addEventListener(type, callback) { events.set(type, callback); },
      emit(type, event = {}) { events.get(type)?.(event); },
      setPointerCapture(id) { this.captures.add(id); },
      hasPointerCapture(id) { return this.captures.has(id); },
      releasePointerCapture(id) { this.captures.delete(id); },
      getContext() { return context; },
    };
  }
  const context = new Proxy({
    createRadialGradient(...args) {
      assert.ok(args.every(Number.isFinite));
      assert.ok(args[2] >= 0 && args[5] >= 0);
      gradients.push(args[5]);
      return { addColorStop() {} };
    },
    drawImage(_image, ...args) { assert.ok(args.every(Number.isFinite)); },
  }, { get: (object, key) => object[key] ?? (() => {}) });
  const elements = new Map();
  const document = Object.assign(element(), {
    hidden: false,
    getElementById(id) {
      if (!elements.has(id)) elements.set(id, element());
      return elements.get(id);
    },
    createElement: element,
  });
  function audioNode(type) {
    const node = { type, connected: false, end: Infinity,
      connect() { this.connected = true; },
      disconnect() { this.connected = false; },
      start() { if (this.buffer && !this.loop) this.end = now / 1000 + this.buffer.duration; },
      stop(time = now / 1000) { this.end = time; },
    };
    for (const key of ['gain', 'frequency', 'Q', 'threshold', 'knee', 'ratio', 'attack', 'release']) {
      node[key] = { value: 0, setValueAtTime() {}, exponentialRampToValueAtTime() {}, setTargetAtTime() {} };
    }
    nodes.push(node);
    return node;
  }
  class AudioContext {
    constructor() { this.state = 'running'; this.sampleRate = 8000; this.destination = {}; }
    get currentTime() { return now / 1000; }
    resume() { this.state = 'running'; return Promise.resolve(); }
    suspend() { this.state = 'suspended'; return Promise.resolve(); }
    createBuffer(_channels, size, rate) {
      const data = new Float32Array(size);
      const buffer = { duration: size / rate, getChannelData: () => data };
      buffers.push(buffer);
      return buffer;
    }
  }
  for (const type of ['Oscillator', 'Gain', 'BiquadFilter', 'BufferSource', 'DynamicsCompressor', 'Convolver']) {
    AudioContext.prototype['create' + type] = () => audioNode(type);
  }
  const window = Object.assign(element(), {
    innerWidth: 390, innerHeight: 844, devicePixelRatio: 3,
    AudioContext: audioAvailable ? AudioContext : undefined,
  });
  const sandbox = {
    window, document, matchMedia: () => ({ matches: mobile }),
    Audio: class {
      constructor() { media.push(this); }
      play() { return Promise.resolve(); }
      pause() { this.paused = true; }
    },
    performance: { now: () => now },
    setTimeout() {}, // Starter balls are irrelevant to deterministic stress cases.
    requestAnimationFrame(callback) { const id = ++nextId; frames.set(id, callback); return id; },
    cancelAnimationFrame(id) { frames.delete(id); },
  };
  const source = readFileSync('static/game/js/game.js', 'utf8').replace(/\}\)\(\);\s*$/, `
    globalThis.inspect = () => ({ balls, particles, impacts, shockwaves, chargeNodes,
      activeSounds, dragging, audioCtx, MAX_BALLS, TRAIL_LEN, dpr });
    globalThis.exercise = { playLaunch, playBounce, playCollision, spawnShockwave };
  })();`);
  vm.runInNewContext(source, sandbox);
  const canvas = document.getElementById('c');
  function pointer(type, x = 100, y = 100, pointerId = 1) {
    canvas.emit(type, { clientX: x, clientY: y, pointerId, isPrimary: pointerId === 1, button: 0 });
  }
  function advance(ms) {
    now += ms;
    for (const node of nodes) {
      if (!node.ended && node.end <= now / 1000) { node.ended = true; node.onended?.(); }
    }
    const callbacks = [...frames.values()];
    frames.clear();
    callbacks.forEach(callback => callback(now));
  }
  return { ...sandbox, canvas, nodes, buffers, media, gradients, frames, pointer, advance };
}

test('mobile stress keeps effects, balls, audio and canvas bounded', () => {
  const g = game();
  for (let i = 0; i < 500; i++) {
    g.pointer('pointerdown');
    g.pointer('pointermove', 5000, 5000);
    g.advance(1000 / 60);
    g.pointer('pointercancel');
    g.pointer('pointerdown');
    g.pointer('pointerup', 10, 10);
    g.exercise.spawnShockwave(100, 100, 0);
    g.advance(1000 / 60);
    const state = g.inspect();
    assert.ok(state.balls.length <= 12);
    assert.ok(state.particles.length <= 250);
    assert.ok(state.impacts.length <= 24);
    assert.ok(state.shockwaves.length <= 2);
    assert.ok(state.activeSounds <= 8);
  }
  assert.ok(g.canvas.width * g.canvas.height <= 1500000);
  assert.ok(g.buffers.length <= 3, 'noise buffers are reused');
  assert.equal(g.media.length, 2, 'media elements are reused');
  assert.equal(g.nodes.filter(n => n.type === 'Convolver').length, 0);
  g.advance(3000);
  assert.ok(g.nodes.filter(n => n.connected).length < 40, 'old audio graphs disconnect');
});

test('secondary touches cannot start or cancel the primary charge', () => {
  const g = game();
  g.pointer('pointerdown');
  const charge = g.inspect().chargeNodes;
  g.pointer('pointerdown', 100, 100, 2);
  g.pointer('pointercancel', 100, 100, 2);
  assert.equal(g.inspect().chargeNodes, charge);
  g.pointer('pointercancel');
  g.advance(200);
  assert.equal(g.inspect().chargeNodes, null);
  assert.equal(g.inspect().balls.length, 1, 'cancel does not launch');
  assert.ok(Object.values(charge).every(node => !node.connected));
});

test('60 Hz and 120 Hz screens simulate the same amount of game time', () => {
  const ageAt = hz => {
    const g = game();
    g.advance(0);
    for (let i = 0; i < hz * 5; i++) g.advance(1000 / hz);
    return g.inspect().balls[0].age;
  };
  assert.equal(ageAt(60), ageAt(120));
});

test('backgrounding cancels charge, audio and animation; resume starts one loop', () => {
  const g = game();
  g.pointer('pointerdown');
  g.document.hidden = true;
  g.document.emit('visibilitychange');
  assert.equal(g.frames.size, 0);
  assert.equal(g.inspect().dragging, false);
  assert.equal(g.inspect().audioCtx.state, 'suspended');
  assert.ok(g.media.every(audio => audio.paused));
  g.advance(60000);
  g.document.hidden = false;
  g.document.emit('visibilitychange');
  g.document.emit('visibilitychange');
  assert.equal(g.frames.size, 1);
  g.advance(16.67);
  assert.ok(g.inspect().balls[0].age <= 2, 'no background catch-up burst');
});

test('desktop retains its ball/trail budgets, full launch synthesis and reverb', () => {
  const g = game(false);
  g.pointer('pointerdown');
  g.pointer('pointercancel');
  for (let i = 0; i < 4; i++) g.exercise.playLaunch(40);
  assert.equal(g.inspect().MAX_BALLS, 20);
  assert.equal(g.inspect().TRAIL_LEN, 60);
  assert.equal(g.inspect().dpr, 2);
  assert.equal(g.nodes.filter(n => n.type === 'Convolver').length, 4);
  assert.equal(g.inspect().activeSounds, 4, 'desktop launches are not throttled');
  g.advance(3000);
  assert.equal(g.nodes.filter(n => n.connected).length, 0);
  assert.equal(g.inspect().activeSounds, 0);
});

test('game is playable without Web Audio support and after a large resize', () => {
  const g = game(true, false);
  g.pointer('pointerdown');
  g.pointer('pointerup', 10, 10);
  assert.equal(g.inspect().balls.length, 2);
  g.window.innerWidth = 3000;
  g.window.innerHeight = 3000;
  g.window.emit('resize');
  assert.ok(g.canvas.width * g.canvas.height <= 1500001);
  g.advance(16.67);
});
