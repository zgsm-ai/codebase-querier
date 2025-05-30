local commands = commandsToExecute
local final_result_path = finalResultPath

-- Print commands to be executed
print("Lua: Commands to execute:")
for i, cmd in ipairs(commands) do
    print("  " .. cmd)
end
print("")

-- Execute commands sequentially
local success = true
local error_message = ""

for i, cmd in ipairs(commands) do
    print("Lua: Executing command [" .. i .. "]: " .. cmd)

    -- Pass only the command to the Go run_command function
    local output, err = run_command(cmd) -- Removed codebasePath argument

    if err then
        print("Lua: Command failed: " .. err)
        error_message = "Command [" .. i .. "] \"" .. cmd .. "\" failed: " .. err
        success = false
        break -- Stop on first error
    else
        print("Lua: Command successful.")
        -- print("Lua: Output:\n" .. output)
    end
end

-- Set global variables to return result to Go
if success then
    print("Lua: All commands executed successfully.")
    print("Lua: Final result path: " .. final_result_path)
    resultPath = final_result_path
    errorMsg = nil -- Explicitly set to nil on success
else
    print("Lua: Command execution failed.")
    resultPath = nil -- Explicitly set to nil on failure
    errorMsg = error_message
end 