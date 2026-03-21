type PasskeyResponse = {
  clientDataJSON: ArrayBuffer;
  attestationObject?: ArrayBuffer;
  authenticatorData?: ArrayBuffer;
  signature?: ArrayBuffer;
  userHandle?: ArrayBuffer | null;
};

type PasskeyCredential = {
  id: string;
  rawId: ArrayBuffer;
  type: string;
  response: PasskeyResponse;
};

export function base64URLToBuffer(value: string): Uint8Array {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const base64 = (value + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  const buffer = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i += 1) {
    buffer[i] = raw.charCodeAt(i);
  }
  return buffer;
}

export function bufferToBase64URL(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function credentialToJSON(credential: Credential | null): Record<string, unknown> | null {
  if (!credential) {
    return null;
  }

  const passkey = credential as unknown as PasskeyCredential;
  const response: Record<string, unknown> = {
    clientDataJSON: bufferToBase64URL(passkey.response.clientDataJSON),
  };

  if (passkey.response.attestationObject) {
    response.attestationObject = bufferToBase64URL(passkey.response.attestationObject);
  }
  if (passkey.response.authenticatorData) {
    response.authenticatorData = bufferToBase64URL(passkey.response.authenticatorData);
  }
  if (passkey.response.signature) {
    response.signature = bufferToBase64URL(passkey.response.signature);
  }
  if (passkey.response.userHandle) {
    response.userHandle = bufferToBase64URL(passkey.response.userHandle);
  }

  return {
    id: passkey.id,
    rawId: bufferToBase64URL(passkey.rawId),
    type: passkey.type,
    response,
  };
}
