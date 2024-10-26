import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as iac from './iac'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack('install')

        await stack.setAllConfig({
            'romeo-install:namespace': { value: core.getInput('namespace') },
            'romeo-install:api-server': { value: core.getInput('api-server') }
        })

        console.info('Deploying Romeo install...')
        const upRes = await stack.up({ onOutput: console.info })

        core.setOutput('kubeconfig', upRes.outputs.kubeconfig.value)
    } catch (error) {
        core.setFailed(`Action failed: ${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    try {
        const stack = await iac.getStack('install')
        await stack.destroy({ onOutput: console.info, remove: true })

        core.info('Romeo environment destroyed.')
    } catch (error) {
        core.warning(`Cleanup failed: ${(error as Error)?.message ?? error}`)
    }
}

// Main
if (!stateHelper.IsPost) {
    void run()
}
// Post
else {
    void cleanup()
}
