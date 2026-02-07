
# Synchrono City Constitution

## Core Principles

### User Sovereignty and Self-Sovereign Identity
- Identity MUST be defined by a Nostr keypair generated and stored only on user
  devices.
- Private keys MUST NEVER be transmitted or escrowed; no operator or authority
  can recover lost keys.
- Loss of a private key MUST result in permanent loss of identity by design.
- Users MUST control their keys, data, and choice of infrastructure.
- The system MUST be robust to any single operator compromise without
  compromising user identity.

Rationale: User autonomy and cryptographic identity are non-negotiable.

### End-to-End Privacy with Untrusted Infrastructure
- Infrastructure MUST be treated as untrusted by default.
- Operators MUST NOT access encrypted message or call content.
- Direct messages MUST be end-to-end encrypted.
- Group calls MUST use cryptographic group key agreement.
- Real-time media MUST be encrypted such that routing infrastructure cannot
  decrypt it.
- The platform MUST NOT support recording; participants may record externally
  and this cannot be prevented.

Rationale: Content privacy is protected even when infrastructure is curious or
compromised.

### Federated Operator Competition
- Synchrono City MUST be federated, not peer-to-peer and not centralized.
- Users MUST be able to choose operators; operators MUST be able to compete on
  trust, performance, and policy.
- No single entity MUST control the network or user identities.

Rationale: Federation preserves choice and resilience.

### Conversation, Not Enforcement
- The platform MUST provide digital coordination signals for conversation and
  meeting, and MUST NOT mediate or verify physical-world interactions.
- Users are responsible for their own safety, judgment, and real-world actions.
- The system MUST facilitate conversation, not enforcement.

Rationale: Human safety and agency cannot be outsourced to infrastructure.

### Resource Accountability and Abuse Resistance
- Actions that impose cost on the network MUST bear proportional cost.
- Proof-of-Work MUST be used to protect shared infrastructure from abuse.

Rationale: Shared infrastructure remains usable under adversarial load.

## Security, Privacy, and Data Boundaries

### Threat Model and Limits
- The system MUST protect conversation content against:
  - Passive network observers.
  - Honest-but-curious infrastructure operators.
  - Other users attempting content surveillance.
- The system MUST NOT claim protection against:
  - Compromised client devices.
  - Participants recording content externally.
  - State-level adversaries with endpoint access or legal compulsion.
  - Traffic analysis and correlation attacks.
  - Malicious operators running modified code.
- No stronger guarantees are implied beyond the above.

### Data Control, Export, and Exit
- Users MAY export at any time: profile metadata, contact and relay lists,
  block and mute lists, and authored content.
- Users MUST NOT be able to export private keys, other users' content, or call
  recordings (not stored).
- Users MAY leave the system at any time by ceasing participation, deleting
  local data, and/or publishing deletion requests to relays.
- Exit MUST NOT require operator permission.

### Location Handling
- Location data MUST NEVER be stored as movement history.
- Maximum permitted precision MUST be geohash level 6 (~1.2 km).
- Location MUST be used only transiently for discovery.
- Clients MUST NOT transmit location data exceeding this precision.
- Infrastructure MAY reject non-conforming data.
- Users in low-density areas accept residual anonymity risk and retain agency
  to reduce precision.

## Infrastructure and Policy Operations

### Community Components
Each community MUST be bound to:
- A Relay (event authority).
- A Sidecar (policy and validation authority).
- A Media Router (SFU) for real-time calls.
- A Media Store for file blobs.

### Sidecar Responsibilities
The Sidecar is the Policy Enforcement Point bridging decentralized identity
with real-time infrastructure. It MUST:
- Authenticate users.
- Issue short-lived access tokens.
- Enforce block lists.
- Manage ephemeral call state.
- Proxy external requests to protect user IP addresses.
- Fail closed if the Sidecar is unavailable.

### Blocking and Interaction Rules
- Client-side blocking MUST be private, encrypted, and local-only.
- Client-side blocking MUST NOT affect presence or participation.
- Public blocking MUST be infrastructure-enforced and prevent direct
  interaction and call entry under defined rules.
- Public blocking MUST be enforced asymmetrically to protect incumbents.
- Operators MAY intervene only based on observable metadata or valid legal
  process.

### Operator Rights and Obligations
- Operators MUST publish jurisdiction and data-handling practices.
- Operators MAY enforce policy using observable metadata.
- Operators MAY impose rate limits or require payment.
- Operators MUST offer at least one privacy-preserving payment method when
  payment is required.
- Operators are solely responsible for compliance with applicable law in their
  jurisdiction.
- Because content is encrypted, operators cannot proactively moderate private
  communications; enforcement is limited to metadata, reports, and lawful
  requests.

## Governance
- This Constitution supersedes all other practices and templates.
- The project follows a Benevolent Dictator for Life (BDFL) model. The steward
  is responsible for maintaining protocol coherence and prioritizing user
  sovereignty and privacy.
- If stewardship becomes unavailable, control passes to a council of
  maintainers selected by demonstrated contribution and judgment.
- At all times, the community retains the right to fork the code and protocol.
- Amendments MUST be documented with rationale, impact, and migration paths for
  breaking changes.
- Every amendment MUST update this document and the Sync Impact Report.
- Every feature spec and plan MUST include a Constitution Check confirming
  compliance with the principles and constraints in this document.
- Compliance reviews MUST occur at design time and before release.
- Semantic versioning MUST be used:
  - MAJOR for backward-incompatible governance or principle changes.
  - MINOR for new principles or materially expanded guidance.
  - PATCH for clarifications and non-semantic refinements.

**Version**: 1.0.0 | **Ratified**: 2026-02-05 | **Last Amended**: 2026-02-05
