import store from '@/store'
import url from '@/utils/url'

export function checkConflict(files, items) {
    if (typeof items === 'undefined' || items === null) {
        items = []
    }

    let folder_upload = files[0].fullPath !== undefined

    let conflict = false
    for (let i = 0; i < files.length; i++) {
        let file = files[i]
        let name = file.name

        if (folder_upload) {
            let dirs = file.fullPath.split("/")
            if (dirs.length > 1) {
                name = dirs[0]
            }
        }

        let res = items.findIndex(function hasConflict(element) {
            return (element.name === this)
        }, name)

        if (res >= 0) {
            conflict = true
            break
        }
    }

    return conflict
}

export function scanFiles(dt) {
    return new Promise((resolve) => {
        let reading = 0
        const contents = []

        if (dt.items !== undefined) {
            for (let item of dt.items) {
                if (item.kind === "file" && typeof item.webkitGetAsEntry === "function") {
                    const entry = item.webkitGetAsEntry()
                    readEntry(entry)
                }
            }
        } else {
            resolve(dt.files)
        }

        function readEntry(entry, directory = "") {
            if (entry.isFile) {
                reading++
                entry.file(file => {
                    reading--

                    file.fullPath = `${directory}${file.name}`
                    contents.push(file)

                    if (reading === 0) {
                        resolve(contents)
                    }
                })
            } else if (entry.isDirectory) {
                const dir = {
                    isDir: true,
                    size: 0,
                    fullPath: `${directory}${entry.name}`
                }

                contents.push(dir)

                readReaderContent(entry.createReader(), `${directory}${entry.name}`)
            }
        }

        function readReaderContent(reader, directory) {
            reading++

            reader.readEntries(function (entries) {
                reading--
                if (entries.length > 0) {
                    for (const entry of entries) {
                        readEntry(entry, `${directory}/`)
                    }

                    readReaderContent(reader, `${directory}/`)
                }

                if (reading === 0) {
                    resolve(contents)
                }
            })
        }
    })
}

export function handleFiles(files, path, overwrite = false) {
    let uuid = store.state.uuid
    if (uuid == '') {
        uuid = uuidv4()
        store.commit('setUUID', uuid)
    }

    for (let i = 0; i < files.length; i++) {
        let file = files[i]

        let filename = (file.fullPath !== undefined) ? file.fullPath : file.name
        let filenameEncoded = url.encodeRFC5987ValueChars(filename)

        let dir = path.replace("files", "").replace("//", "/")
        if (filename.indexOf("/") != -1) {
            dir += filename.split("/")[0]
        }
        if (dir[dir.length - 1] == "/") {
            dir = dir.substr(0, dir.length - 1)
        }

        dir = url.encodeRFC5987ValueChars(dir)
        let id = store.state.upload.id

        let itemPath = path + filenameEncoded

        if (file.isDir) {
            itemPath = path
            let folders = file.fullPath.split("/")

            for (let i = 0; i < folders.length; i++) {
                let folder = folders[i]
                let folderEncoded = encodeURIComponent(folder)
                itemPath += folderEncoded + "/"
            }
        }

        const item = {
            id,
            path: itemPath,
            file,
            uuid,
            overwrite,
            dir,
        }

        store.dispatch('upload/upload', item);
    }
}

function uuidv4() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}