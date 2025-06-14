import * as fs from 'fs'
import * as path from 'path'
import * as os from 'os'

function expandHome(filePath: string): string {
    if (filePath.startsWith('~')) {
        return path.join(os.homedir(), filePath.slice(1))
    }
    return filePath
}

export function resolveInput(input: string): string {
    const maybePath = path.resolve(expandHome(input))

    if (fs.existsSync(maybePath) && fs.statSync(maybePath).isFile()) {
        console.log('Loading kubeconfig from file: %s', input)
        return fs.readFileSync(maybePath, 'utf-8')
    }

    console.log('Using kubeconfig as it')
    return input
}
