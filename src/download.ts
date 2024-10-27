import * as core from '@actions/core'
import { Buffer } from 'buffer'
import * as path from 'path'
import * as fs from 'fs'
import * as yauzl from 'yauzl'

void import('node-fetch')

type Coverout = {
    merged: string
}

async function run(): Promise<void> {
    try {
        const server = core.getInput('server')
        const response = await fetch(server + '/coverout')
        const data = (await response.json()) as Coverout

        await extract(data.merged, core.getInput('directory'))
    } catch (error) {
        core.setFailed(`${(error as Error)?.message ?? error}`)
    }
}

async function extract(merged: string, outputDir: string): Promise<void> {
    // 1. Decode base64
    const decoded = Buffer.from(merged, 'base64')
    core.info(decoded.toString('utf8'))

    // 2. Unzip content
    yauzl.fromBuffer(decoded, { lazyEntries: true }, (err, zipFile) => {
        if (err) throw err

        if (!fs.existsSync(outputDir)) {
            fs.mkdirSync(outputDir, { recursive: true })
        }

        zipFile.readEntry()
        zipFile.on('entry', (entry: yauzl.Entry) => {
            const filePath = path.join(outputDir, entry.fileName)

            if (entry.fileName.endsWith('/')) {
                // Create directory if not exist
                fs.mkdirSync(filePath, { recursive: true })
                zipFile.readEntry()
            } else {
                // Extract and save file
                zipFile.openReadStream(entry, (suberr, readStream) => {
                    if (suberr) throw suberr

                    fs.mkdirSync(path.dirname(filePath), { recursive: true })

                    const writeStream = fs.createWriteStream(filePath)
                    readStream.pipe(writeStream)
                    readStream.on('end', () => zipFile.readEntry())
                })
            }
        })
    })
}

// Main
void run()
