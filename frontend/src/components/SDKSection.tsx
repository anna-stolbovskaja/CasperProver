export default function SDKSection() {
  return (
    <section id="sdk" className="py-24 bg-cp-card/30">
      <div className="cp-section">
        <div className="text-center mb-14">
          <p className="text-xs font-mono text-red-500 tracking-widest mb-3">INTEGRATION</p>
          <h2 className="text-3xl font-extrabold text-white mb-4">SDK and MCP server.</h2>
          <p className="text-gray-500 max-w-lg mx-auto">Two ways to integrate proof generation into your stack.</p>
        </div>

        <div className="grid lg:grid-cols-2 gap-6 max-w-5xl mx-auto">
          {/* Go SDK */}
          <div className="bg-cp-card rounded-2xl border border-cp-border overflow-hidden">
            <div className="p-6 border-b border-cp-border">
              <h3 className="text-white font-bold mb-1">Go SDK</h3>
              <p className="text-sm text-gray-500">Native Go client for proof generation and verification.</p>
              <div className="mt-3 bg-black/40 rounded-lg px-4 py-2 font-mono text-xs text-gray-400 inline-block">
                go get github.com/anna-stolbovskaja/CasperProver/sdk
              </div>
            </div>
            <pre className="p-5 text-xs font-mono text-gray-300 overflow-x-auto leading-relaxed"><code>{`import "github.com/anna-stolbovskaja/CasperProver/sdk"

client := sdk.New("https://casperprover-api.onrender.com")
proof, err := client.CreateProof(sdk.ProofInput{
    Agent:  "agent-alpha",
    Input:  "loan_decision",
    Output: "approved",
    Model:  "gpt-4",
})
fmt.Println(proof.ProofHash) // 0x7f3a...`}</code></pre>
          </div>

          {/* MCP */}
          <div className="bg-cp-card rounded-2xl border border-cp-border overflow-hidden">
            <div className="p-6 border-b border-cp-border">
              <h3 className="text-white font-bold mb-1">MCP Server</h3>
              <p className="text-sm text-gray-500">Connect any AI agent to proof generation via MCP.</p>
              <div className="mt-3 bg-black/40 rounded-lg px-4 py-2 font-mono text-xs text-gray-400 inline-block">
                go run ./cmd/mcp-server
              </div>
            </div>
            <pre className="p-5 text-xs font-mono text-gray-300 overflow-x-auto leading-relaxed"><code>{`// claude_desktop_config.json
{
  "mcpServers": {
    "casper-prover": {
      "command": "casper-prover-mcp",
      "env": {
        "PROVER_API": "https://casperprover-api.onrender.com"
      }
    }
  }
}
// Tools: create_proof, verify_proof, list_proofs`}</code></pre>
          </div>
        </div>
      </div>
    </section>
  )
}
