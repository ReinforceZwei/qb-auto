/**
 * Schema-driven definitions of the qb-auto custom API endpoints.
 * Each endpoint renders a purpose-built form in the playground page.
 */

export type FieldValue = string | number | boolean;

export interface EndpointField {
  name: string;
  label: string;
  /** Where the value lands in the request. */
  location: 'path' | 'query' | 'body';
  type: 'text' | 'number' | 'boolean' | 'select';
  required?: boolean;
  /** Optional rule: the field is required only when this returns true. */
  requiredWhen?: (values: Record<string, FieldValue>) => boolean;
  description?: string;
  placeholder?: string;
  options?: { value: string; label: string }[];
  defaultValue?: FieldValue;
}

export interface EndpointDef {
  id: string;
  label: string;
  method: 'GET' | 'POST';
  /** Path template; `{name}` placeholders map to path fields. */
  path: string;
  description: string;
  fields: EndpointField[];
}

export const ENDPOINTS: EndpointDef[] = [
  {
    id: 'torrent-complete',
    label: 'Create job from torrent',
    method: 'GET',
    path: '/api/torrent-complete',
    description:
      'Registers a completed torrent as a job. Anime-category jobs run the title + rsync pipeline; everything else goes straight to the notification webhook. Re-sending an errored hash retries it.',
    fields: [
      {
        name: 'hash',
        location: 'query',
        type: 'text',
        label: 'Torrent hash',
        required: true,
        placeholder: 'e.g. 9f8e… (40-char info hash)',
      },
      {
        name: 'category',
        location: 'query',
        type: 'text',
        label: 'Category',
        placeholder: 'e.g. anime',
      },
    ],
  },
  {
    id: 'resolve-anime-title',
    label: 'Resolve anime title',
    method: 'POST',
    path: '/api/resolve-anime-title',
    description:
      'Resolves a torrent folder name into an anime title using the LLM + TMDb (with an optional Wikipedia fallback).',
    fields: [
      {
        name: 'folder_name',
        location: 'body',
        type: 'text',
        label: 'Folder name',
        required: true,
        placeholder: 'e.g. [SubsPlease] Oshi no Ko (1080p)',
      },
      {
        name: 'search_anime_list',
        location: 'body',
        type: 'boolean',
        label: 'Search anime list',
        defaultValue: false,
        description:
          'Also look up the matching anime-list record so the response includes an anime_list_id.',
      },
    ],
  },
  {
    id: 'job-retry',
    label: 'Restart a failed job',
    method: 'POST',
    path: '/api/jobs/{id}/retry',
    description:
      'Re-queues a failed job. Full retry re-runs the normal pipeline; skip-to-rsync uses your manual title and goes straight to the transfer stage (anime jobs only).',
    fields: [
      {
        name: 'id',
        location: 'path',
        type: 'text',
        label: 'Job ID',
        required: true,
        placeholder: 'e.g. abcdefghijklmno',
      },
      {
        name: 'mode',
        location: 'body',
        type: 'select',
        label: 'Mode',
        required: true,
        defaultValue: 'full',
        options: [
          { value: 'full', label: 'Full retry — normal pipeline' },
          { value: 'rsync', label: 'Skip to rsync — manual title' },
        ],
      },
      {
        name: 'anime_title',
        location: 'body',
        type: 'text',
        label: 'Anime title',
        requiredWhen: (values) => values.mode === 'rsync',
        description: 'Required when skipping straight to rsync.',
        placeholder: 'e.g. My Hero Academia S3',
      },
      {
        name: 'anime_list_id',
        location: 'body',
        type: 'text',
        label: 'AnimeList ID',
        description: 'Optional — leave empty to skip marking the anime as downloaded.',
      },
      {
        name: 'tmdb_id',
        location: 'body',
        type: 'number',
        label: 'TMDb ID',
        description: 'Optional reference info.',
      },
      {
        name: 'tmdb_season',
        location: 'body',
        type: 'number',
        label: 'Season number',
        description: 'Optional reference info.',
      },
    ],
  },
];

/** Returns the default value for a field. */
export function fieldDefault(field: EndpointField): FieldValue {
  switch (field.type) {
    case 'boolean':
      return field.defaultValue ?? false;
    case 'select':
      return field.defaultValue ?? field.options?.[0]?.value ?? '';
    default:
      return field.defaultValue ?? '';
  }
}

/** Initial values for an endpoint's fields. */
export function initValues(def: EndpointDef): Record<string, FieldValue> {
  return Object.fromEntries(def.fields.map((f) => [f.name, fieldDefault(f)]));
}

/** Assembles the request URL (path params substituted, query params appended). */
export function buildRequestUrl(
  def: EndpointDef,
  values: Record<string, FieldValue>,
): string {
  let url = def.path;
  for (const f of def.fields) {
    if (f.location === 'path') {
      url = url.replace(`{${f.name}}`, encodeURIComponent(String(values[f.name] ?? '')));
    }
  }

  const query = def.fields
    .filter((f) => f.location === 'query')
    .map((f) => [f.name, values[f.name]] as const)
    .filter(([, v]) => v !== '' && v !== undefined && v !== null)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join('&');

  return query ? `${url}?${query}` : url;
}

/** Assembles the JSON body from body fields, omitting empty/optional values. */
export function buildBody(
  def: EndpointDef,
  values: Record<string, FieldValue>,
): string | undefined {
  const bodyFields = def.fields.filter((f) => f.location === 'body');
  if (bodyFields.length === 0) return undefined;

  const obj: Record<string, unknown> = {};
  for (const f of bodyFields) {
    const v = values[f.name];
    if (f.type === 'boolean') {
      obj[f.name] = Boolean(v);
      continue;
    }
    if (v === '' || v === undefined || v === null) continue;
    obj[f.name] = v;
  }

  return Object.keys(obj).length > 0 ? JSON.stringify(obj, null, 2) : undefined;
}

/** Validates required fields (including conditional requiredWhen rules). */
export function validateFields(
  def: EndpointDef,
  values: Record<string, FieldValue>,
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const f of def.fields) {
    const isRequired = f.required || (f.requiredWhen ? f.requiredWhen(values) : false);
    if (!isRequired) continue;
    const v = values[f.name];
    if (v === '' || v === undefined || v === null) {
      errors[f.name] = 'Required';
    }
  }
  return errors;
}
