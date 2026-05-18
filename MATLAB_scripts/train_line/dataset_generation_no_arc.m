% environment setup
model_name = 'model';
load_system(model_name);       
total = 200;

% path to the fault-mode block
fault_block = [model_name '/Three-Phase Fault'];

% list of fault types to iterate over
fault_types = {'AG', 'BG', 'CG', 'ABG', 'BCG', 'CAG', 'ABCG', 'AB', 'BC', 'CA', 'ABC'};

% check that base data folders exist and create them if needed
if ~exist('./data/rms', 'dir'), mkdir('./data/rms'); end
if ~exist('./data/sv_and_trip', 'dir'), mkdir('./data/sv_and_trip'); end

% fault generation every kilometer iterate from 1 km to 199 km to avoid division by zero in line blocks
for fault_dist = 1:1:199
    
    % calculate line variables
    part1 = fault_dist;
    part2 = total - fault_dist;
    
    % nested loop over fault types
    for i = 1:length(fault_types)
        current_fault = fault_types{i};
        
        if ~exist(['./data/rms/' current_fault], 'dir'), mkdir(['./data/rms/' current_fault]); end
        if ~exist(['./data/sv_and_trip/' current_fault], 'dir'), mkdir(['./data/sv_and_trip/' current_fault]); end
        
        switch current_fault
            % single-phase-to-ground
            case 'AG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'off', 'GroundFault', 'on');
            case 'BG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'on');
            case 'CG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'on');
            
            % two-phase-to-ground
            case 'ABG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'on');
            case 'BCG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'on');
            case 'CAG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'on');
            
            % three-phase-to-ground                    
            case 'ABCG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'on');
                        
            % two-phase
            case 'AB'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'off');
            case 'BC'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'off');
            case 'CA'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'off');
            
            % three-phase                   
            case 'ABC'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'off');
              
        end
        
        % print progress to the console
        fprintf('\nFault simulation %d km, type %s...', fault_dist, current_fault);
        
        % run simulation - 'SrcWorkspace','current' lets sim see the part1 and part2 variables
        simOut = sim(model_name, 'SrcWorkspace', 'current');
        
        
        % build unique file names
        t_stamp = string(datetime('now'), 'ddMMyyyy_HHmmss');
        
        filepath_rms = sprintf('./data/rms/%s/%s_%03d_km.csv', current_fault, t_stamp, fault_dist);
        filepath_sv_trip = sprintf('./data/sv_and_trip/%s/%s_%03d_km.csv', current_fault, t_stamp, fault_dist);
        
        % save data
        writematrix(simOut.I_U_ABCN_RMS, filepath_rms, 'Delimiter', ';');
        writematrix(simOut.SV_and_Trip, filepath_sv_trip, 'Delimiter', ';');
    end
    
end

fprintf('Dataset generation completed!\n');
