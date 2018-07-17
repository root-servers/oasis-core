//! Contract call processing service.
use std::sync::Arc;

use grpcio;

use ekiden_compute_api::{Contract, SubmitTxRequest, SubmitTxResponse};
use ekiden_core::contract::batch::CallBatch;
use ekiden_core::futures::prelude::*;

use super::super::consensus::ConsensusFrontend;

struct ContractServiceInner {
    /// Consensus frontend.
    consensus_frontend: Arc<ConsensusFrontend>,
}

#[derive(Clone)]
pub struct ContractService {
    inner: Arc<ContractServiceInner>,
}

impl ContractService {
    /// Create new compute server instance.
    pub fn new(consensus_frontend: Arc<ConsensusFrontend>) -> Self {
        ContractService {
            inner: Arc::new(ContractServiceInner { consensus_frontend }),
        }
    }
}

impl Contract for ContractService {
    fn submit_tx(
        &self,
        ctx: grpcio::RpcContext,
        mut request: SubmitTxRequest,
        sink: grpcio::UnarySink<SubmitTxResponse>,
    ) {
        measure_histogram_timer!("submit_tx_time");
        measure_counter_inc!("submit_tx_calls");

        let batch = CallBatch(vec![request.take_data()]);

        self.inner.consensus_frontend.append_batch(batch);

        ctx.spawn(sink.success(SubmitTxResponse::new()).map_err(|_error| ()));
    }
}
