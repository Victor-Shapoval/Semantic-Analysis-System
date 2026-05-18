% environment setup
model_name = 'model';
load_system(model_name);       
total = 200;

% path to the fault-mode block
fault_block = [model_name '/Three-Phase Fault'];

% list of fault types to iterate over
fault_types = {'AG', 'BG', 'CG', 'ABG', 'BCG', 'CAG', 'ABCG', 'AB', 'BC', 'CA', 'ABC'};

% parameters for resistance calculation (Warrington + PUE)
L_arc = 2.5;           % arc length in meters
R_ground_value = 10;   % tower grounding resistance (Ohm)

% check that base folders exist
if ~exist('./data/rms', 'dir'), mkdir('./data/rms'); end
if ~exist('./data/sv_and_trip', 'dir'), mkdir('./data/sv_and_trip'); end

% generation: every kilometer for maximum dataset accuracy
for fault_dist = 1:1:199
    
    % calculate line sections for Distributed Parameters Line blocks
    part1 = fault_dist;
    part2 = total - fault_dist;
    
    for i = 1:length(fault_types)
        current_fault = fault_types{i};
        
        % create subfolders for fault types
        if ~exist(['./data/rms/' current_fault], 'dir'), mkdir(['./data/rms/' current_fault]); end
        if ~exist(['./data/sv_and_trip/' current_fault], 'dir'), mkdir(['./data/sv_and_trip/' current_fault]); end
        
        % define fault logic
        is_ground_fault = contains(current_fault, 'G');
        % phase-to-phase faults (including three-phase faults), where the method requires sqrt(3)
        is_multi_phase = (length(current_fault) >= 2 && ~is_ground_fault) || strcmp(current_fault, 'ABC');
        
        switch current_fault
            case 'AG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'off', 'GroundFault', 'on');
            case 'BG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'on');
            case 'CG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'on');
            case 'ABG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'on');
            case 'BCG'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'on');
            case 'CAG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'on');
            case 'ABCG'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'on');
            case 'AB'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'off', 'GroundFault', 'off');
            case 'BC'
                set_param(fault_block, 'FaultA', 'off', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'off');
            case 'CA'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'off', 'FaultC', 'on', 'GroundFault', 'off');
            case 'ABC'
                set_param(fault_block, 'FaultA', 'on', 'FaultB', 'on', 'FaultC', 'on', 'GroundFault', 'off');
        end
        
        % arc resistance calculation
        % preliminary run to measure current (bolted mode)
        set_param(fault_block, 'FaultResistance', '0.001', 'GroundResistance', '0.001');
        tempSim = sim(model_name, 'SrcWorkspace', 'current');
        
        % find the steady-state current in the fault interval (0.55 - 0.6 s)
        time_vec = tempSim.tout;
        fault_indices = find(time_vec >= 0.55 & time_vec <= 0.6);
        
        if isempty(fault_indices)
            % if the interval is not found, use the maximum over the whole time range
            I_bolted = max(max(tempSim.I_U_ABCN_RMS(:, [1, 3, 5])));
        else
            % use the maximum during the steady-state fault interval
            I_bolted = max(max(tempSim.I_U_ABCN_RMS(fault_indices, [1, 3, 5]))); 
        end
        
        % apply sqrt(3) to phase-to-phase faults according to the method
        if is_multi_phase
             I_arc_calc = sqrt(3) * I_bolted; 
        else
             I_arc_calc = I_bolted; 
        end
        
        % Warrington formula
        R_arc = (28700 * L_arc) / (I_arc_calc^1.4);
        if R_arc > 50 || isnan(R_arc), R_arc = 50; end
        
        % final simulation with calculated parameters
        set_param(fault_block, 'FaultResistance', num2str(R_arc));
        if is_ground_fault
            set_param(fault_block, 'GroundResistance', num2str(R_ground_value));
        else
            set_param(fault_block, 'GroundResistance', '0.001');
        end
        
        fprintf('\nProgress: %d km, type %s (r_arc = %.3f Ohm)', fault_dist, current_fault, R_arc);
        
        simOut = sim(model_name, 'SrcWorkspace', 'current');
        
        % save data
        t_stamp = string(datetime('now'), 'ddMMyyyy_HHmmss');
        f_name_rms = sprintf('./data/rms/%s/%s_%03dkm.csv', current_fault, t_stamp, fault_dist);
        f_name_sv = sprintf('./data/sv_and_trip/%s/%s_%03dkm.csv', current_fault, t_stamp, fault_dist);
        
        writematrix(simOut.I_U_ABCN_RMS, f_name_rms, 'Delimiter', ';');
        writematrix(simOut.SV_and_Trip, f_name_sv, 'Delimiter', ';');
    end
end
fprintf('\nDataset generation completed successfully!\n');
