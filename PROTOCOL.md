# Synchrono City Protocol Specification

**Version:** 1.0.0 (Canonical)

> This document defines the *technical law of the system*.
> Where the Constitution defines **rights and constraints**, this specification defines **mechanisms, formats, and required behavior**.
>
> In case of conflict, the **Constitution is authoritative**.

---

## 1. Scope and Authority

This specification defines:

* Required Nostr NIPs
* Event kinds and semantics
* Cryptographic requirements
* Proof‑of‑Work rules
* Sidecar API contracts
* Enforcement boundaries

This document is **normative**. Implementations claiming Synchrono City compatibility MUST comply.

---

## 2. Protocol Foundations

Synchrono City is built on **Nostr** with mandatory extensions.

### 2.1 Required NIPs

Implementations MUST support:

* NIP‑01 — Core protocol
* NIP‑02 — Contact lists
* NIP‑09 — Event deletion
* NIP‑10 — Threading
* NIP‑13 — Proof of Work
* NIP‑17 — Private messages
* NIP‑29 — Groups
* NIP‑42 — Relay authentication
* NIP‑44 — Encryption (XChaCha20‑Poly1305)
* NIP‑51 — Lists (mute, block)
* NIP‑59 — Gift Wraps
* NIP‑65 — Relay lists
* NIP‑78 — Application data
* NIP‑98 — HTTP authentication
* B7 — Blossom media storage

Optional NIPs MAY be supported but MUST NOT degrade interoperability.

---

## 3. Event Model

### 3.1 Event Validation (All Clients & Relays)

All received events MUST be validated in this order:

1. Signature validity
2. Timestamp bounds (±5 minutes)
3. Required tags present
4. Proof‑of‑Work target met (if applicable)
5. Expiration tag (if present)

Invalid events MUST be discarded.

---

## 4. Event Kinds (Canonical)

### 4.1 Core

| Kind | Purpose          |
| ---- | ---------------- |
| 0    | Metadata         |
| 1    | Short text note  |
| 5    | Deletion request |
| 1059 | Gift Wrap        |

### 4.2 Groups (NIP‑29)

| Kind  | Purpose        |
| ----- | -------------- |
| 39000 | Group metadata |
| 39001 | Group admins   |
| 39002 | Group members  |
| 39003 | Group roles    |
| 9000  | Put user       |
| 9001  | Remove user    |
| 9002  | Edit metadata  |
| 9003  | Create role    |
| 9004  | Delete role    |
| 9005  | Delete event   |
| 9007  | Create group   |
| 9008  | Delete group   |
| 9009  | Create invite  |
| 9021  | Join request   |
| 9022  | Leave request  |

### 4.3 Calls (Persistent)

| Kind  | Purpose       |
| ----- | ------------- |
| 1020  | Call start    |
| 1021  | Call end      |
| 1022 | DM call offer |
| 1023 | DM call end   |

### 4.4 Calls (Ephemeral)

| Kind  | Purpose                    |
| ----- | -------------------------- |
| 20002 | Join request               |
| 20003 | Token response (HTTP only) |
| 20004 | Participant joined         |
| 20005 | Participant left           |
| 20007 | Epoch leader transfer      |
| 20011 | DM call answer             |
| 20012 | DM call reject             |
| 20020 | MLS welcome                |
| 20021 | MLS commit                 |

---

## 5. Proof of Work

Proof‑of‑Work enforces **resource asymmetry**.

| Action             | Kind          | Bits |
| ------------------ | ------------- | ---- |
| Create group       | 9007          | 28   |
| Call start         | 1020          | 24   |
| Profile update     | 0             | 20   |
| MLS key package    | 30022         | 16   |
| Join call          | 20002         | 12   |
| Block list update  | 10006         | 12   |
| DM answer / reject | 20011 / 20012 | 8    |

Relays and Sidecars MUST reject insufficient PoW.

---

## 6. Location Encoding

* Maximum precision: **geohash level 6**
* Coordinates MUST be truncated to match geohash precision
* Location data MUST NOT be stored as movement history

---

## 7. Encryption Model

### 7.1 Direct Messages

* NIP‑44 encryption
* End‑to‑end

### 7.2 Group Calls

* MLS (RFC 9420)
* First participant becomes **Epoch Leader**
* Leader issues commits for joins and leaves
* Leadership transfers automatically on departure or failure

Clients MUST detect unauthorized keys ("ghost devices") and alert users.

### 7.3 Media Encryption

* Media frames encrypted via insertable streams
* Keys derived from MLS exporter secrets
* Routing infrastructure cannot decrypt media

---

## 8. Sidecar Protocol

The Sidecar is the **Policy Enforcement Point**.

### 8.1 Responsibilities

* Validate membership
* Enforce block lists
* Issue short‑lived access tokens
* Manage ephemeral MLS state
* Proxy external requests to hide user IPs

### 8.2 Authentication

All Sidecar requests MUST use **NIP‑98 HTTP Auth**.

### 8.3 Failure Semantics

If the Sidecar is unavailable:

* No new calls may start
* No new participants may join
* Existing calls may continue until natural termination

---

## 9. Block and Mute Semantics

### 9.1 Mute (Kind 10000)

* Encrypted
* Client‑side only
* Content hidden locally
* No infrastructure enforcement

### 9.2 Block (Kind 10006)

* Public
* Infrastructure‑enforced
* Prevents DMs and call entry
* First‑arriver rule protects incumbents

---

## 10. Sidecar API (Abstract)

Endpoints are implementation‑defined but MUST support:

* `/health`
* `/token/group`
* `/token/dm`
* `/proxy`
* `/mls/state/{room}`
* `/mls/commit`

Responses MUST be authenticated, time‑bounded, and non‑replayable.

---

## 11. Time and Clock Rules

* Clients SHOULD warn at ±30 seconds drift
* Clients MAY refuse to create events at ±5 minutes
* Relays and Sidecars MUST reject events beyond ±5 minutes

---

## 12. Compliance

Implementations MUST:

* Validate all inputs
* Clear sensitive material from memory
* Never log decrypted content or tokens
* Enforce protocol limits consistently

---

## 13. Extensibility

Extensions are permitted if they:

* Do not break compatibility
* Are clearly identified as non‑standard
* Do not weaken privacy or security

Proprietary extensions that fragment interoperability are prohibited.

---

## 14. Versioning

* Semantic versioning is used
* Breaking changes require migration paths
* Deprecated features require notice

---

**End of Protocol Specification**
