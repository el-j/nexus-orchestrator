import { NexusClient } from '../src/submit';
import { HttpClient } from '@actions/http-client';

// Mock @actions/http-client
jest.mock('@actions/http-client');

const MockHttpClient = HttpClient as jest.MockedClass<typeof HttpClient>;

describe('NexusClient', () => {
  let client: NexusClient;
  let mockPostJson: jest.Mock;
  let mockGetJson: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    mockPostJson = jest.fn();
    mockGetJson = jest.fn();
    MockHttpClient.prototype.postJson = mockPostJson;
    MockHttpClient.prototype.getJson = mockGetJson;
    client = new NexusClient('http://127.0.0.1:63987');
  });

  describe('submitTask', () => {
    it('returns task_id and status on 201', async () => {
      mockPostJson.mockResolvedValue({
        statusCode: 201,
        result: { task_id: 'abc-123', status: 'QUEUED' },
        headers: {},
      });

      const res = await client.submitTask({
        projectPath: '/project',
        instruction: 'Do the thing',
      });

      expect(res.task_id).toBe('abc-123');
      expect(res.status).toBe('QUEUED');
      expect(mockPostJson).toHaveBeenCalledWith(
        'http://127.0.0.1:63987/api/tasks',
        expect.objectContaining({ instruction: 'Do the thing' }),
      );
    });

    it('throws on non-201 status', async () => {
      mockPostJson.mockResolvedValue({
        statusCode: 500,
        result: null,
        headers: {},
      });

      await expect(client.submitTask({ projectPath: '/p', instruction: 'x' })).rejects.toThrow(
        'HTTP 500',
      );
    });

    it('reports "?" when the response has no status code', async () => {
      mockPostJson.mockResolvedValue({ result: null, headers: {} });
      await expect(client.submitTask({ projectPath: '/p', instruction: 'x' })).rejects.toThrow(
        'HTTP ?',
      );
    });

    it('throws on null result', async () => {
      mockPostJson.mockResolvedValue({ statusCode: 201, result: null, headers: {} });
      await expect(client.submitTask({ projectPath: '/p', instruction: 'x' })).rejects.toThrow(
        'Empty response',
      );
    });
  });

  describe('getTask', () => {
    it('returns task data on success', async () => {
      mockGetJson.mockResolvedValue({
        statusCode: 200,
        result: {
          id: 'abc-123',
          status: 'COMPLETED',
          logs: 'done',
          projectPath: '/p',
          targetFile: '',
          instruction: 'x',
        },
        headers: {},
      });

      const task = await client.getTask('abc-123');
      expect(task.status).toBe('COMPLETED');
      expect(task.id).toBe('abc-123');
    });

    it('throws when result is null', async () => {
      mockGetJson.mockResolvedValue({ statusCode: 404, result: null, headers: {} });
      await expect(client.getTask('missing')).rejects.toThrow('not found');
    });
  });

  describe('waitForTask', () => {
    it('returns immediately on terminal status', async () => {
      mockGetJson.mockResolvedValue({
        statusCode: 200,
        result: {
          id: 't1',
          status: 'COMPLETED',
          logs: 'ok',
          projectPath: '/p',
          targetFile: '',
          instruction: 'x',
        },
        headers: {},
      });

      const task = await client.waitForTask('t1', 30_000);
      expect(task.status).toBe('COMPLETED');
      expect(mockGetJson).toHaveBeenCalledTimes(1);
    });

    it('polls until terminal status', async () => {
      mockGetJson
        .mockResolvedValueOnce({
          statusCode: 200,
          result: {
            id: 't2',
            status: 'QUEUED',
            logs: '',
            projectPath: '',
            targetFile: '',
            instruction: '',
          },
          headers: {},
        })
        .mockResolvedValueOnce({
          statusCode: 200,
          result: {
            id: 't2',
            status: 'PROCESSING',
            logs: '',
            projectPath: '',
            targetFile: '',
            instruction: '',
          },
          headers: {},
        })
        .mockResolvedValueOnce({
          statusCode: 200,
          result: {
            id: 't2',
            status: 'COMPLETED',
            logs: 'done',
            projectPath: '',
            targetFile: '',
            instruction: '',
          },
          headers: {},
        });

      jest.useFakeTimers();
      const waitPromise = client.waitForTask('t2', 60_000);
      // Advance past each 5s sleep between polls
      await jest.advanceTimersByTimeAsync(5_000);
      await jest.advanceTimersByTimeAsync(5_000);
      await jest.advanceTimersByTimeAsync(5_000);
      jest.useRealTimers();

      const task = await waitPromise;
      expect(task.status).toBe('COMPLETED');
    });

    it('throws on timeout', async () => {
      mockGetJson.mockResolvedValue({
        statusCode: 200,
        result: {
          id: 'slow',
          status: 'QUEUED',
          logs: '',
          projectPath: '',
          targetFile: '',
          instruction: '',
        },
        headers: {},
      });

      jest.useFakeTimers();
      const waitPromise = client.waitForTask('slow', 100);
      // Register the rejection handler BEFORE advancing timers, so the
      // rejection is not "unhandled" when the timer fires the throw.
      const assertion = expect(waitPromise).rejects.toThrow('did not complete');
      // Advance past the 5s sleep so the loop re-evaluates the deadline (which is 100ms)
      await jest.advanceTimersByTimeAsync(6_000);
      jest.useRealTimers();
      await assertion;
    });
  });

  describe('api token and waiting tasks', () => {
    const task = (status: string, extra: Record<string, unknown> = {}) => ({
      statusCode: 200,
      result: {
        id: 'w',
        status,
        logs: '',
        projectPath: '',
        targetFile: '',
        instruction: '',
        ...extra,
      },
      headers: {},
    });

    it('sends the API token as a bearer header when one is given, and none otherwise', () => {
      const lastOptions = () =>
        (HttpClient as unknown as { lastOptions: Record<string, unknown> }).lastOptions;
      new NexusClient('http://127.0.0.1:63987', 'secret');
      expect(lastOptions()).toMatchObject({ headers: { Authorization: 'Bearer secret' } });
      new NexusClient('http://127.0.0.1:63987');
      expect(lastOptions()).not.toHaveProperty('headers');
    });

    it('says when a WAITING task resumes, and names it in the timeout error', async () => {
      const info = jest.spyOn(require('@actions/core'), 'info').mockImplementation(() => undefined);
      mockGetJson.mockResolvedValue(task('WAITING', { waitUntil: '2026-06-01T12:00:00Z' }));
      jest.useFakeTimers();
      const waitPromise = client.waitForTask('w', 100);
      const assertion = expect(waitPromise).rejects.toThrow(
        'last status: WAITING, resumes 2026-06-01T12:00:00Z',
      );
      await jest.advanceTimersByTimeAsync(6_000);
      jest.useRealTimers();
      await assertion;
      expect(info).toHaveBeenCalledWith(
        expect.stringContaining('waiting until 2026-06-01T12:00:00Z'),
      );
      info.mockRestore();
    });

    it('keeps the plain timeout text for a task that does not wait', async () => {
      mockGetJson.mockResolvedValue(task('NO_PROVIDER'));
      jest.useFakeTimers();
      const waitPromise = client.waitForTask('w', 100);
      const assertion = expect(waitPromise).rejects.toThrow('(last status: NO_PROVIDER)');
      await jest.advanceTimersByTimeAsync(6_000);
      jest.useRealTimers();
      await assertion;
    });
  });
});
