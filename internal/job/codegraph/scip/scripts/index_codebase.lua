-- This script executes a list of commands provided by the Go code.
-- It reports the final result path or an error message back to Go.

-- commandsToExecute is a Lua table (list of strings) set by the Go code.
local commands_to_execute = commandsToExecute

-- finalResultPath is a string set by the Go code, the expected path of the index file on success.
local final_result_path = finalResultPath

-- Global variables to report result back to Go.
resultPath = nil
errorMsg = nil

-- --- Execute Commands --- --
if not commands_to_execute or #commands_to_execute == 0 then
    errorMsg = "No commands provided by Go to execute."
else
    local execution_error = nil
    -- Execute the determined commands
    for i, cmd_to_execute in ipairs(commands_to_execute) do
        -- Call the Go-provided function to run the command
        -- run_command returns output string, error_string (nil if no error)
        local output, err_str = run_command(cmd_to_execute)

        if err_str then
            execution_error = "Command failed: '" .. tostring(cmd_to_execute) .. "'\nError: " .. tostring(err_str) .. "\nOutput: \n" .. tostring(output) -- Added newline for output and ensured strings
            break -- Stop executing further commands on failure
        end
        -- Optional: print or log output of successful commands for debugging
        -- print("Command '" .. tostring(cmd_to_execute) .. "' output:\n" .. tostring(output))
     end

    if execution_error then
        errorMsg = "Command execution failed: " .. execution_error
    else
        -- All commands executed successfully, set the final result path
        resultPath = final_result_path
    end
end

-- The Go code will read resultPath or errorMsg after the script finishes.