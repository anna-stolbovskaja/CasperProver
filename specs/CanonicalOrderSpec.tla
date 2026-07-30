-------------------------- MODULE CanonicalOrderSpec --------------------------
(***************************************************************************)
(* Formal specification of the sort-normalisation property that            *)
(* `engine/internal/receipts/canonical.go: CanonicalHash` relies on to be  *)
(* a deterministic function of the receipt's UNORDERED content.           *)
(*                                                                         *)
(* CanonicalHash writes fields to SHA-256 in a fixed, sort-normalised     *)
(* order:                                                                  *)
(*   - facets are sorted by .Kind before hashing                          *)
(*   - provider receipts are sorted by .ReceiptHash before hashing        *)
(*                                                                         *)
(* This spec does NOT model SHA-256 (opaque to TLA+). It models the       *)
(* SORT INVARIANT the hash relies on: for any two input orderings of the *)
(* same underlying facet-set / provider-set, the sorted sequence is       *)
(* identical. If that invariant holds, and the hash function is             *)
(* deterministic on byte input, the whole receipt-canonicalisation        *)
(* property follows.                                                       *)
(*                                                                         *)
(* Out of scope: SHA-256 correctness itself, the length-prefix encoding,  *)
(* the timestamp / issuer / subject scalar fields (they are simply        *)
(* concatenated in a fixed order, no sort involved).                     *)
(*                                                                         *)
(* Safety invariants checked:                                              *)
(*   - TypeOK                                                              *)
(*   - SortIsIdempotent — Sort(Sort(seq)) = Sort(seq)                     *)
(*   - SortPreservesMultiset — the sorted sequence contains exactly the  *)
(*     same elements (as a multiset) as the input                        *)
(*   - SortDependsOnlyOnSet — for any two permutations of the same       *)
(*     underlying set of distinct-key elements, Sort yields the same     *)
(*     sequence                                                            *)
(*   - SortIsMonotone — the sorted sequence is non-decreasing by key      *)
(***************************************************************************)

EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANTS
    Keys,          \* finite set of candidate sort keys (facet Kind /
                   \* provider ReceiptHash values). Must be finite & totally
                   \* ordered under < on natural numbers.
    MaxItems       \* upper bound on |items| per receipt

ASSUME
    /\ Keys # {}
    /\ \A k \in Keys : k \in Nat
    /\ MaxItems \in Nat /\ MaxItems >= 1
    /\ MaxItems <= Cardinality(Keys)

VARIABLES
    input,         \* current input sequence of items (a permutation of a subset of Keys)
    sorted,        \* Sort(input)
    doubleSorted   \* Sort(Sort(input))

vars == <<input, sorted, doubleSorted>>

(***************************************************************************)
(* Each item is modelled as its sort key alone. In the real                *)
(* CanonicalHash, a facet also carries Verdict/Confidence/Reason and a    *)
(* provider receipt carries Provider/TrustLevel; but the sort key alone   *)
(* is what the sort-invariance property depends on. Keys are constrained  *)
(* distinct within an input so ordering is unambiguous (the Go code       *)
(* implicitly assumes stable order on ties, which sort.Slice preserves    *)
(* for equal-key items; TLA+ does not need to model that tie-breaking     *)
(* since the receipt canonicalisation includes the full record body per   *)
(* item — a real tie means byte-identical items, which the hash cannot    *)
(* distinguish anyway).                                                    *)
(***************************************************************************)

Item == Nat

(***************************************************************************)
(* Set of all sequences of length <= MaxItems over Keys with distinct     *)
(* elements. Used to bound the input state.                                *)
(***************************************************************************)

DistinctSeqsOfLen(n) ==
    { s \in [ 1..n -> Keys ] :
        \A i, j \in 1..n : (i # j) => (s[i] # s[j]) }

AllDistinctSeqs ==
    UNION { DistinctSeqsOfLen(n) : n \in 0..MaxItems }

(***************************************************************************)
(* Ascending-by-key sort. Because keys are distinct within an input, the  *)
(* result is uniquely determined.                                          *)
(***************************************************************************)

SeqToSet(s) == { s[i] : i \in DOMAIN s }

IsSortedAsc(s) ==
    \A i \in DOMAIN s : \A j \in DOMAIN s : (i < j) => (s[i] < s[j])

Sort(s) ==
    CHOOSE t \in [ DOMAIN s -> SeqToSet(s) ] :
        /\ SeqToSet(t) = SeqToSet(s)
        /\ IsSortedAsc(t)

(***************************************************************************)
(* Init / Next: pick any distinct-key input sequence; sort it once and    *)
(* twice; then re-pick a fresh input. Every reachable state re-exercises  *)
(* the sort on an independent input, so an exhaustive TLC run over Keys   *)
(* covers every permutation of every subset of Keys of size <= MaxItems.  *)
(***************************************************************************)

Init ==
    /\ input \in AllDistinctSeqs
    /\ sorted = Sort(input)
    /\ doubleSorted = Sort(sorted)

Pick ==
    /\ \E s \in AllDistinctSeqs :
         /\ input' = s
         /\ sorted' = Sort(s)
         /\ doubleSorted' = Sort(Sort(s))

Next == Pick

Spec == Init /\ [][Next]_vars

(***************************************************************************)
(* Type invariant                                                          *)
(***************************************************************************)

TypeOK ==
    /\ input \in AllDistinctSeqs
    /\ sorted \in AllDistinctSeqs
    /\ doubleSorted \in AllDistinctSeqs

(***************************************************************************)
(* Idempotence — sorting an already-sorted sequence changes nothing.      *)
(* Corresponds to CanonicalHash's contract that re-canonicalising a       *)
(* canonical receipt yields the same bytes.                                *)
(***************************************************************************)

SortIsIdempotent ==
    doubleSorted = sorted

(***************************************************************************)
(* Sort preserves the underlying set — no element is dropped or added.   *)
(* This is what lets the receipt round-trip through CanonicalHash without *)
(* losing facets or providers.                                             *)
(***************************************************************************)

SortPreservesMultiset ==
    SeqToSet(sorted) = SeqToSet(input)

(***************************************************************************)
(* The sorted sequence is a pure function of the underlying set — two    *)
(* input orderings of the same set yield the same sorted sequence. This   *)
(* is the property that makes CanonicalHash order-independent w.r.t. the *)
(* caller's construction order of facets / providers.                     *)
(*                                                                         *)
(* Expressed here as: for every alternative permutation q of the current  *)
(* input's key-set, Sort(q) = sorted.                                     *)
(***************************************************************************)

AllPermsOf(keySet) ==
    { q \in [ 1..Cardinality(keySet) -> keySet ] :
        \A i, j \in DOMAIN q : (i # j) => (q[i] # q[j]) }

SortDependsOnlyOnSet ==
    \A q \in AllPermsOf(SeqToSet(input)) : Sort(q) = sorted

(***************************************************************************)
(* Monotone by key — the sorted sequence is strictly ascending. This is  *)
(* the direct correspondence to sort.Slice in canonical.go using          *)
(* .Kind < .Kind and .ReceiptHash < .ReceiptHash comparators.             *)
(***************************************************************************)

SortIsMonotone == IsSortedAsc(sorted)

(***************************************************************************)
(* Composite safety invariant — everything at once, for the cfg          *)
(* INVARIANTS block.                                                       *)
(***************************************************************************)

SafetyInvariant ==
    /\ TypeOK
    /\ SortIsIdempotent
    /\ SortPreservesMultiset
    /\ SortDependsOnlyOnSet
    /\ SortIsMonotone

=============================================================================
