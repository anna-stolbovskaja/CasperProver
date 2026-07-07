#![no_std]
#![no_main]

// Session code (NOT a stored contract) that atomically deposits a stake
// into the stake-slashing contract: gets the caller's own main purse
// (only available in session context, not inside a called contract),
// transfers `amount` into the target contract's purse, then calls
// record_stake in the SAME deploy. Bundling these means there is never a
// window where funds moved but the stake wasn't recorded, or vice versa.

extern crate alloc;

use casper_contract::contract_api::{account, runtime, system};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::contracts::ContractHash;
use casper_types::{RuntimeArgs, U512};

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let stake_slashing_contract: ContractHash = runtime::get_named_arg("stake_slashing_contract");
    let amount: U512 = runtime::get_named_arg("amount");

    let main_purse = account::get_main_purse();

    let contract_purse = runtime::call_contract(
        stake_slashing_contract,
        "get_purse",
        RuntimeArgs::new(),
    );

    system::transfer_from_purse_to_purse(main_purse, contract_purse, amount, None)
        .unwrap_or_revert();

    let mut record_args = RuntimeArgs::new();
    record_args.insert("amount", amount).unwrap_or_revert();
    runtime::call_contract::<()>(stake_slashing_contract, "record_stake", record_args);
}
