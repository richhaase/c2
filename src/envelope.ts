export interface Envelope<T> {
  schema: string;
  generated_at: string;
  data: T;
}

export function envelope<T>(schema: string, data: T, now?: Date): Envelope<T> {
  return {
    schema,
    generated_at: (now ?? new Date()).toISOString(),
    data,
  };
}

export function printJSON<T>(schema: string, data: T): void {
  console.log(JSON.stringify(envelope(schema, data), null, 2));
}
