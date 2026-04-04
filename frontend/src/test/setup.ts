import { vi } from 'vitest'

class MockEventSource {
  url: string
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  onerror: ((event?: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
  }

  close() {}
}

vi.stubGlobal('EventSource', MockEventSource)