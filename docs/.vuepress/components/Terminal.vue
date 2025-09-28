<template>
  <div class="terminal">
    <div class="terminal__header">
      <div class="terminal__traffic">
        <span class="terminal__dot terminal__dot--close" aria-hidden="true"></span>
        <span class="terminal__dot terminal__dot--min" aria-hidden="true"></span>
        <span class="terminal__dot terminal__dot--max" aria-hidden="true"></span>
      </div>
      <div class="terminal__title">{{ computedTitle }}</div>
      <button
        class="terminal__copy"
        type="button"
        :aria-label="copied ? 'Copied' : 'Copy terminal contents'"
        @click="copyContent"
      >
        <span v-if="!copied" class="icon-copy" aria-hidden="true">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
            <path d="M16 1H4c-1.1 0-2 .9-2 2v12h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
          </svg>
        </span>
        <span v-else class="icon-check" aria-hidden="true">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
          </svg>
        </span>
      </button>
    </div>
    <div class="terminal__body" ref="bodyRef">
      <div ref="slotRef" v-show="!renderedText">
        <slot />
      </div>
      <template v-if="renderedText">
        <div v-for="(line, i) in parsedLines" :key="i" class="term-line" :class="{ 'term-line--cmd': line.type === 'command', 'term-line--out': line.type === 'output', 'term-line--blank': line.type === 'blank', 'term-line--comment': line.type === 'comment' }">
          <template v-if="line.type === 'command'">
            <span class="term-prompt">{{ line.prompt }}</span>
            <button class="term-copy-btn" type="button" :title="'Copy'" :aria-label="'Copy: ' + line.command" @click="copyOnly(line.command)">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">
                <path d="M16 1H4c-1.1 0-2 .9-2 2v12h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
              </svg>
            </button>
            <span class="term-cmd">
              <template v-for="(seg, si) in tokenizeCmd(line.command)" :key="si">
                <span :class="seg.class">{{ seg.text }}</span>
              </template>
            </span>
          </template>
          <template v-else-if="line.type === 'comment'">
            <span class="term-comment">{{ line.raw }}</span>
          </template>
          <template v-else-if="line.type === 'output'">
            <span class="term-out">{{ line.raw }}</span>
          </template>
          <template v-else>
            <br />
          </template>
        </div>
      </template>
    </div>
  </div>

</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';

const props = defineProps<{
  title?: string
}>()

const computedTitle = computed(() => (props.title && props.title.trim()) ? props.title : 'Terminal')

const copied = ref(false)
const bodyRef = ref<HTMLElement | null>(null)
const slotRef = ref<HTMLElement | null>(null)
const renderedText = ref<string | null>(null)

function copyContent() {
  const el = bodyRef.value
  if (!el) return
  const text = (renderedText.value ?? el.innerText) || ''
  if (!text.trim()) return
  navigator.clipboard.writeText(text).then(() => {
    copied.value = true
    window.setTimeout(() => (copied.value = false), 1200)
  })
}

function scrollToBottom() {
  const el = bodyRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

onMounted(() => {
  nextTick(() => {
    const s = slotRef.value
    if (s) {
      // Prefer content of fenced code blocks rendered as <pre><code>
      const code = s.querySelector('pre code') as HTMLElement | null
      let text = code?.textContent || (s as HTMLElement).innerText || ''
      // Normalize endings, decode nbsp, trim trailing spaces
      text = text
        .replace(/\r\n/g, "\n")
        .replace(/\r/g, "\n")
        .replace(/\u00A0/g, ' ')
        .replace(/\s+$/,'')
      // Drop leading/trailing blank lines and normalize common indentation so prompts align
      let lines = text.split("\n")
      while (lines.length && !lines[0].trim()) lines.shift()
      while (lines.length && !lines[lines.length-1].trim()) lines.pop()
      // Compute common leading spaces across non-blank lines
      let common = Infinity
      for (const ln of lines) {
        if (!ln.trim()) continue
        const m = ln.match(/^ +/)
        const spaces = m ? m[0].length : 0
        common = Math.min(common, spaces)
      }
      if (common !== Infinity && common > 0) {
        lines = lines.map(ln => ln.startsWith(' '.repeat(common)) ? ln.slice(common) : ln)
      }
      renderedText.value = lines.join("\n")
    }
    scrollToBottom()
  })
})

const parsedLines = computed(() => {
  const raw = renderedText.value ?? ''
  if (!raw) return [] as Array<{ type: 'command' | 'output' | 'blank' | 'comment'; raw: string; prompt?: string; command?: string; comment?: string }>
  const lines = raw.split(/\r?\n/)
  const result: Array<{ type: 'command' | 'output' | 'blank' | 'comment'; raw: string; prompt?: string; command?: string; comment?: string }> = []

  for (const line of lines) {
    if (!line.trim()) {
      result.push({ type: 'blank', raw: '' })
      continue
    }

    // Detect comments (lines starting with #)
    const commentMatch = line.match(/^(\s*#\s*)(.*)$/)
    if (commentMatch) {
      result.push({ type: 'comment', raw: line, comment: commentMatch[2] })
      continue
    }

    // Detect prompts with a command (no splitting of inline output; display line as-is)
    let m = line.match(/^(\s*\([^)]*\)\s*\$\s+)(.+)$/) // (env) $ cmd
    if (!m) m = line.match(/^(\s*\$\s+)(.+)$/) // $ cmd
    if (!m) m = line.match(/^(\s*PS>\s+)(.+)$/) // PS> cmd (PowerShell)

    if (m) {
      const prompt = m[1]
      let after = m[2]
      // Only add blank line before commands if the previous line is not a comment or blank
      const lastLine = result[result.length - 1]
      if (result.length > 0 && lastLine && lastLine.type !== 'blank' && lastLine.type !== 'comment') {
        result.push({ type: 'blank', raw: '' })
      }
      result.push({ type: 'command', raw: line, prompt, command: after })
      continue
    }
    result.push({ type: 'output', raw: line })
  }
  return result
})

function copyOnly(text: string | undefined) {
  if (!text) return
  navigator.clipboard.writeText(text)
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

// Tokenize without using innerHTML replacement to avoid corrupting text
function tokenizeCmd(cmd?: string): Array<{ text: string; class?: string }> {
  if (!cmd) return []
  const segments: Array<{ text: string; class?: string }> = []
  let i = 0

  // helper to push plain text
  const pushText = (t: string, cls?: string) => {
    if (!t) return
    segments.push({ text: t, class: cls })
  }

  // detect initial binary/command
  const leading = cmd.match(/^\s*([\w.\-\/]+)/)
  if (leading) {
    const leadingText = cmd.slice(0, leading.index! + leading[0].length)
    const pre = leadingText.slice(0, leadingText.length - leading[1].length)
    if (pre) pushText(pre)
    pushText(leading[1], 'hl-cmd')
    i = leading.index! + leading[0].length
  }

  while (i < cmd.length) {
    const ch = cmd[i]
    // quoted string
    if (ch === '"' || ch === '\'') {
      const quote = ch
      let j = i + 1
      while (j < cmd.length && cmd[j] !== quote) j++
      pushText(cmd.slice(i, Math.min(j + 1, cmd.length)), 'hl-str')
      i = Math.min(j + 1, cmd.length)
      continue
    }
    // operators: |, ||, >, >>, 2>
    const opMatch = cmd.slice(i).match(/^(\|\||\||>>?|2>)/)
    if (opMatch) {
      pushText(opMatch[0], 'hl-op')
      i += opMatch[0].length
      continue
    }
    // flags: -x, --long, with optional =value
    // Only match flags that are preceded by whitespace or at start, and followed by whitespace, =, or end
    const flagMatch = cmd.slice(i).match(/^(\s+)((?:--[\w-]+|-[a-zA-Z])(?:=[^\s]+)?)(?=\s|$)/)
    if (flagMatch) {
      pushText(flagMatch[1], undefined) // whitespace
      pushText(flagMatch[2], 'hl-flag') // flag
      i += flagMatch[0].length
      continue
    }
    // Handle flags at the very beginning of the command
    if (i === 0) {
      const startFlagMatch = cmd.match(/^((?:--[\w-]+|-[a-zA-Z])(?:=[^\s]+)?)(?=\s|$)/)
      if (startFlagMatch) {
        pushText(startFlagMatch[1], 'hl-flag')
        i += startFlagMatch[1].length
        continue
      }
    }
    // default: consume one character
    pushText(cmd[i])
    i++
  }

  return segments
}

watch(bodyRef, () => {
  scrollToBottom()
})
</script>

<style scoped>
.terminal {
  background-color: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-bg-soft);
  border-radius: 12px;
  transition: border-color var(--vp-t-color), background-color var(--vp-t-color);
}

.terminal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--vp-c-default-soft);
}

.terminal__traffic {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal__dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--vp-c-default-soft);
}
.terminal__dot--close { background-color: var(--vp-c-danger-2, #ff5f56); }
.terminal__dot--min   { background-color: var(--vp-c-warning-2, #ffbd2e); }
.terminal__dot--max   { background-color: var(--vp-c-success-2, #27c93f); }

.terminal__title {
  flex: 1 1 auto;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.terminal__copy {
  appearance: none;
  border: 1px solid var(--vp-c-brand-1);
  background: transparent;
  color: var(--vp-c-brand-1);
  padding: 4px 6px;
  border-radius: 6px;
  cursor: pointer;
}
.terminal__copy:hover,
.terminal__copy:focus {
  background: var(--vp-c-brand-1);
  color: var(--vp-c-accent-text);
}

.terminal__body {
  padding: 14px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.terminal__pre {
  margin: 0;
  white-space: pre-wrap;
}

.term-line {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: 0;
}
.term-line + .term-line { margin-top: 6px; }

.term-line--cmd .term-prompt {
  color: var(--vp-c-text-2);
}
.term-cmd { color: var(--vp-c-text-1); white-space: pre-wrap; }
.term-out { color: var(--vp-c-text-1); white-space: pre-wrap; }
.term-comment { color: var(--vp-c-text-3); white-space: pre-wrap; font-style: italic; }
.term-blank { color: transparent; }

.term-copy-btn {
  appearance: none;
  border: 1px solid var(--vp-c-brand-1);
  background: transparent;
  color: var(--vp-c-brand-1);
  padding: 2px 4px;
  border-radius: 6px;
  cursor: pointer;
  margin-left: -2px;
}
.term-copy-btn:hover,
.term-copy-btn:focus { background: var(--vp-c-brand-1); color: var(--vp-c-accent-text); }

/* Syntax highlighting tokens */
.term-cmd :deep(.hl-cmd) { color: var(--vp-c-brand-1); font-weight: 600; }
.term-cmd :deep(.hl-flag) { color: var(--vp-c-text-2); }
.term-cmd :deep(.hl-str) { color: var(--vp-c-warning-1, #e6a700); }
.term-cmd :deep(.hl-op) { color: var(--vp-c-default-3, #888); }

/* Normalize code block rendering inside the terminal body */
.terminal__body :deep(pre) {
  margin: 0;
  padding: 0;
  background: transparent !important;
  border: none;
}
.terminal__body :deep(code) {
  background: transparent !important;
  border: none;
}

/* Make plain Markdown paragraphs look like terminal lines */
.terminal__body :deep(p) {
  margin: 0;
}
.terminal__body :deep(p + p) {
  margin-top: 6px;
}

/* Optional prompt styling if users include plain lines rather than fenced code */
.terminal__body :deep(.prompt) {
  color: var(--vp-c-text-2);
}
</style>
