-- debpack-lsp.lua
-- Neovim companion plugin for debpack-lsp.
--
-- Installation (packer):
--
--   use {
--     "BAMF0/debpack-lsp",
--     run = "make install",
--     config = function() require("debpack-lsp").setup() end,
--   }
--
-- Installation (lazy.nvim):
--
--   { "BAMF0/debpack-lsp", build = "make install", config = true }
--
-- The plugin uses vim.lsp.start directly (no nvim-lspconfig required).

local M = {}

-- Debian-specific file type patterns used to trigger the LSP.
-- Each entry: { pattern = <glob>, ft = <filetype hint> }
local DEBIAN_PATTERNS = {
  "*/debian/control",
  "*/debian/changelog",
  "*/debian/rules",
  "*/debian/watch",
  "*/debian/copyright",
  "*/debian/patches/*",
  "*/debian/*.install",
  "*/debian/*.dirs",
  "*/debian/*.docs",
  "*/debian/*.links",
  "*/debian/*.manpages",
}

-- Find the root of the source package (the directory containing debian/).
local function find_root(fname)
  return vim.fs.root(fname, { "debian" })
end

-- Locate the debpack-lsp binary.
local function find_binary()
  local bin = vim.fn.exepath("debpack-lsp")
  if bin ~= "" then
    return bin
  end
  -- Fallback: look in the repo root next to lua/ (useful during development).
  -- __FILE__ is lua/debpack-lsp.lua, so repo root is one level up.
  local here = debug.getinfo(1, "S").source:sub(2)          -- strip leading @
  local repo_root = vim.fs.dirname(vim.fs.dirname(here))    -- lua/ -> repo root
  local sibling = vim.fs.joinpath(repo_root, "debpack-lsp")
  if vim.uv.fs_stat(sibling) then
    return sibling
  end
  return nil
end

-- Start (or reuse) the LSP client for the current buffer.
local function attach(args)
  local buf = args.buf
  local fname = vim.api.nvim_buf_get_name(buf)
  if fname == "" then return end

  local root = find_root(fname)
  if not root then return end

  local bin = find_binary()
  if not bin then
    vim.notify(
      "debpack-lsp: binary not found. Install it with:\n  go install github.com/BAMF0/debpack-lsp@latest",
      vim.log.levels.WARN
    )
    return
  end

  vim.lsp.start({
    name    = "debpack-lsp",
    cmd     = { bin },
    root_dir = root,
    capabilities = vim.lsp.protocol.make_client_capabilities(),
    -- Pass the buffer so start() can check if a client is already running.
  }, { bufnr = buf, reuse_client = function(client, _)
    return client.name == "debpack-lsp" and client.config.root_dir == root
  end })
end

function M.setup(opts)
  opts = opts or {}

  local group = vim.api.nvim_create_augroup("debpack_lsp", { clear = true })

  -- Attach on BufReadPost / BufNewFile for any file matching debian/ patterns.
  vim.api.nvim_create_autocmd({ "BufReadPost", "BufNewFile" }, {
    group    = group,
    pattern  = DEBIAN_PATTERNS,
    callback = attach,
    desc     = "Attach debpack-lsp to debian/ files",
  })

  -- Set helpful file-type overrides so Neovim syntax-highlights them.
  local ft_map = {
    ["*/debian/control"]   = "debcontrol",
    ["*/debian/changelog"] = "debchangelog",
    ["*/debian/copyright"] = "debcopyright",
    ["*/debian/rules"]     = "make",
  }
  for pattern, ft in pairs(ft_map) do
    vim.filetype.add({ pattern = { [pattern] = ft } })
  end
end

return M
