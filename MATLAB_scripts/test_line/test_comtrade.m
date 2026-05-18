% =========================================================================
% Test script for line 3: 220 kV, 150 km, AC 240/32
% 11 fault types x 3 positions x 2 modes (bolted + with arc) = 66 scenarios
% Save to COMTRADE (IEEE C37.111-1999)
% =========================================================================

script_dir = fileparts(mfilename('fullpath'));
addpath(fullfile(script_dir, '..'));

model_name = 'model';
if bdIsLoaded(model_name), bdclose(model_name); end
load_system(fullfile(script_dir, model_name));

% line parameters 3
total = 150;                   % line length, km
line_name = 'Line3_220kV';
U_nom_kV = 220;

% arc transition resistance (from calculations)
R_d = 2.344;                   % Ohm
R_ground = 10;                 % tower grounding resistance, Ohm

% path to the fault-mode block
fault_block = [model_name '/Three-Phase Fault'];

% fault types
fault_types = {'AG', 'ABG', 'ABCG', 'CA', 'ABC'};

% 4 positions: 5%, 75%, 85%, 98% of line length
positions = [round(total*0.05), round(total*0.10), round(total*0.15), round(total*0.20), round(total*0.25), round(total*0.30), round(total*0.35), round(total*0.40), round(total*0.45), round(total*0.50), round(total*0.55), round(total*0.60), round(total*0.65), round(total*0.70), round(total*0.75), round(total*0.80), round(total*0.85), round(total*0.90), round(total*0.95)];
pos_names = {'5pct', '10pct', '15pct', '20pct', '25pct', '30pct', '35pct', '40pct', '45pct', '50pct', '55pct', '60pct', '65pct', '70pct', '75pct', '80pct', '85pct', '90pct', '95pct'};

% 2 modes resistance: bolted fault and fault with arc transition resistance
mode_names   = {'metallic', 'arc'};
R_fault_vals = [0.001,       R_d];

% output directory (relative to the script location)
% directory is created inside the mode loop


n_total = length(mode_names) * length(positions) * length(fault_types);
fprintf('=== Line 3: %d kV, %d km, R_d = %.3f Ohm ===\n', U_nom_kV, total, R_d);
fprintf('Fault positions: %s\n', mat2str(positions));
fprintf('Total scenarios: %d\n\n', n_total);

counter = 0;

for m = 1:length(mode_names)
    current_mode = mode_names{m};
    R_fault = R_fault_vals(m);

    for p = 1:length(positions)
        fault_dist = positions(p);
        part_1 = fault_dist;
        part_2 = total - fault_dist;

        for i = 1:length(fault_types)
            current_fault = fault_types{i};
            is_ground = contains(current_fault, 'G');

            % configure the Fault block
            switch current_fault
                case 'AG',   set_param(fault_block,'FaultA','on', 'FaultB','off','FaultC','off','GroundFault','on');
                case 'BG',   set_param(fault_block,'FaultA','off','FaultB','on', 'FaultC','off','GroundFault','on');
                case 'CG',   set_param(fault_block,'FaultA','off','FaultB','off','FaultC','on', 'GroundFault','on');
                case 'ABG',  set_param(fault_block,'FaultA','on', 'FaultB','on', 'FaultC','off','GroundFault','on');
                case 'BCG',  set_param(fault_block,'FaultA','off','FaultB','on', 'FaultC','on', 'GroundFault','on');
                case 'CAG',  set_param(fault_block,'FaultA','on', 'FaultB','off','FaultC','on', 'GroundFault','on');
                case 'ABCG', set_param(fault_block,'FaultA','on', 'FaultB','on', 'FaultC','on', 'GroundFault','on');
                case 'AB',   set_param(fault_block,'FaultA','on', 'FaultB','on', 'FaultC','off','GroundFault','off');
                case 'BC',   set_param(fault_block,'FaultA','off','FaultB','on', 'FaultC','on', 'GroundFault','off');
                case 'CA',   set_param(fault_block,'FaultA','on', 'FaultB','off','FaultC','on', 'GroundFault','off');
                case 'ABC',  set_param(fault_block,'FaultA','on', 'FaultB','on', 'FaultC','on', 'GroundFault','off');
            end

            set_param(fault_block, 'FaultResistance', num2str(R_fault));
            if is_ground && m == 2
                set_param(fault_block, 'GroundResistance', num2str(R_ground));
            else
                set_param(fault_block, 'GroundResistance', '0.001');
            end

            counter = counter + 1;
            fprintf('[%d/%d] %s | %s | %s (%d km) | R_f=%.3f Ohm\n', ...
                counter, n_total, current_mode, current_fault, pos_names{p}, fault_dist, R_fault);

            simOut = sim(model_name, 'SrcWorkspace', 'current');

            out_dir = fullfile(script_dir, 'data', current_mode);
            if ~exist(out_dir, 'dir'), mkdir(out_dir); end
            fname = fullfile(out_dir, sprintf('%s_%s_%03dkm', current_fault, pos_names{p}, fault_dist));
            write_comtrade(fname, simOut.tout, simOut.SV_and_Trip, line_name, current_fault, fault_dist, 50);
        end
    end
end

fprintf('\nLine 3: generation %d COMTRADE files completed!\n', n_total);
