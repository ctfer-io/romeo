import * as core from '@actions/core'
import * as stateHelper from './state-helper'
import * as fs from './fs'
import * as iac from './iac'

const dst = 'install'

async function run(): Promise<void> {
    try {
        const stack = await iac.getStack(dst, dst)

        await stack.setAllConfig({
            'install:kubeconfig': {
                value: fs.resolveInput(core.getInput('kubeconfig')),
                secret: true
            },
            'install:namespace': {
                value: core.getInput('namespace')
            },
            'install:api-server': {
                value: core.getInput('api-server')
            },
            'install:harden': {
                value: core.getInput('harden', { required: false })
            }
        })

        const upRes = await stack.up({ onOutput: core.info })

        core.setOutput('kubeconfig', upRes.outputs['kubeconfig'].value)
        core.setOutput('namespace', upRes.outputs['namespace'].value)
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

async function cleanup(): Promise<void> {
    try {
        const stack = await iac.getStack(dst, dst)
        await stack.destroy({ onOutput: core.info, remove: true })
    } catch (error) {
        core.warning(`${(error as Error)?.message ?? error}`)
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
