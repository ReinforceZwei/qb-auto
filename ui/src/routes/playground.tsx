import { useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Card,
  Code,
  Divider,
  Group,
  NumberInput,
  Select,
  Stack,
  Switch,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { IconSend } from '@tabler/icons-react';
import { playgroundRequest, type PlaygroundResult } from '../lib/api';
import {
  ENDPOINTS,
  buildBody,
  buildRequestUrl,
  initValues,
  validateFields,
  type EndpointDef,
  type EndpointField,
  type FieldValue,
} from '../lib/playground';

interface FieldInputProps {
  field: EndpointField;
  value: FieldValue;
  error?: string;
  onChange: (value: FieldValue) => void;
}

function FieldInput({ field, value, error, onChange }: FieldInputProps) {
  switch (field.type) {
    case 'boolean':
      return (
        <Switch
          label={field.label}
          description={field.description}
          checked={Boolean(value)}
          onChange={(e) => onChange(e.currentTarget.checked)}
        />
      );
    case 'number':
      return (
        <NumberInput
          label={field.label}
          description={field.description}
          placeholder={field.placeholder}
          error={error}
          value={value as number | ''}
          onChange={(v) => onChange(typeof v === 'number' ? v : '')}
        />
      );
    case 'select':
      return (
        <Select
          label={field.label}
          description={field.description}
          data={field.options ?? []}
          value={String(value)}
          onChange={(v) => onChange(v ?? '')}
          allowDeselect={false}
        />
      );
    default:
      return (
        <TextInput
          label={field.label}
          description={field.description}
          placeholder={field.placeholder}
          required={field.required}
          error={error}
          value={String(value)}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      );
  }
}

function formatResponse(data: unknown): string {
  if (typeof data === 'string') return data;
  try {
    return JSON.stringify(data, null, 2);
  } catch {
    return String(data);
  }
}

export function PlaygroundPage() {
  const [endpoint, setEndpoint] = useState<EndpointDef>(ENDPOINTS[0]);
  const [values, setValues] = useState<Record<string, FieldValue>>(() =>
    initValues(ENDPOINTS[0]),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<PlaygroundResult | null>(null);

  const selectEndpoint = (id: string | null) => {
    const def = ENDPOINTS.find((e) => e.id === id);
    if (!def) return;
    setEndpoint(def);
    setValues(initValues(def));
    setErrors({});
    setResult(null);
  };

  const setValue = (name: string, value: FieldValue) => {
    setValues((prev) => ({ ...prev, [name]: value }));
  };

  const url = useMemo(() => buildRequestUrl(endpoint, values), [endpoint, values]);
  const body = useMemo(() => buildBody(endpoint, values), [endpoint, values]);
  const hasBody = body !== undefined;

  const pathFields = endpoint.fields.filter((f) => f.location === 'path');
  const queryFields = endpoint.fields.filter((f) => f.location === 'query');
  const bodyFields = endpoint.fields.filter((f) => f.location === 'body');

  const handleSend = async () => {
    const errs = validateFields(endpoint, values);
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setSending(true);
    try {
      const res = await playgroundRequest(endpoint.method, url, hasBody ? body : undefined);
      setResult(res);
    } catch (err) {
      setResult({
        ok: false,
        status: 0,
        data: err instanceof Error ? err.message : 'Request failed',
        durationMs: 0,
      });
    } finally {
      setSending(false);
    }
  };

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>API playground</Title>
        <Text c="dimmed" size="sm">
          Purpose-built forms for the qb-auto API. Pick an endpoint, fill in the
          fields, and send.
        </Text>
      </div>

      <Card withBorder radius="md" padding="lg">
        <Stack gap="md">
          <Select
            label="Endpoint"
            data={ENDPOINTS.map((e) => ({
              value: e.id,
              label: `${e.method} ${e.label}`,
            }))}
            value={endpoint.id}
            onChange={selectEndpoint}
          />

          <Group gap="xs">
            <Badge
              color={endpoint.method === 'GET' ? 'green' : 'indigo'}
              variant="light"
            >
              {endpoint.method}
            </Badge>
            <Text ff="monospace" size="sm" c="dimmed">
              {endpoint.path}
            </Text>
          </Group>
          <Text size="sm" c="dimmed">
            {endpoint.description}
          </Text>

          <Divider />

          {pathFields.length > 0 && (
            <>
              <Text size="xs" fw={600} c="dimmed" tt="uppercase">
                Path parameters
              </Text>
              {pathFields.map((f) => (
                <FieldInput
                  key={f.name}
                  field={f}
                  value={values[f.name]}
                  error={errors[f.name]}
                  onChange={(v) => setValue(f.name, v)}
                />
              ))}
            </>
          )}

          {queryFields.length > 0 && (
            <>
              <Text size="xs" fw={600} c="dimmed" tt="uppercase">
                Query parameters
              </Text>
              {queryFields.map((f) => (
                <FieldInput
                  key={f.name}
                  field={f}
                  value={values[f.name]}
                  error={errors[f.name]}
                  onChange={(v) => setValue(f.name, v)}
                />
              ))}
            </>
          )}

          {bodyFields.length > 0 && (
            <>
              <Text size="xs" fw={600} c="dimmed" tt="uppercase">
                Request body
              </Text>
              {bodyFields.map((f) => (
                <FieldInput
                  key={f.name}
                  field={f}
                  value={values[f.name]}
                  error={errors[f.name]}
                  onChange={(v) => setValue(f.name, v)}
                />
              ))}
            </>
          )}

          <Divider />

          <Group justify="space-between" align="center" wrap="nowrap">
            <Group gap="xs" style={{ minWidth: 0 }}>
              <Badge
                color={endpoint.method === 'GET' ? 'green' : 'indigo'}
                variant="light"
                size="sm"
              >
                {endpoint.method}
              </Badge>
              <Text ff="monospace" size="sm" truncate>
                {url}
              </Text>
            </Group>
            <Button
              leftSection={<IconSend size={16} />}
              onClick={handleSend}
              loading={sending}
            >
              Send
            </Button>
          </Group>

          {hasBody && (
            <Code block style={{ maxHeight: 240, overflow: 'auto' }}>
              {body}
            </Code>
          )}
        </Stack>
      </Card>

      {result && (
        <Card withBorder radius="md" padding="lg">
          <Stack gap="sm">
            <Group gap="sm">
              <Badge color={result.ok ? 'green' : 'red'} variant="light">
                {result.status === 0 ? 'Error' : result.status}
              </Badge>
              <Text size="sm" c="dimmed">
                {result.durationMs} ms
              </Text>
            </Group>
            <Code block style={{ maxHeight: 480, overflow: 'auto' }}>
              {formatResponse(result.data)}
            </Code>
          </Stack>
        </Card>
      )}
    </Stack>
  );
}
